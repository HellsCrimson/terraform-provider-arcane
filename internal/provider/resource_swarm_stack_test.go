package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// swarmStackSchema returns the resource schema for building tfsdk State/Plan in tests.
func swarmStackSchema(t *testing.T) tfsdk.State {
	t.Helper()
	var sresp resource.SchemaResponse
	(&SwarmStackResource{}).Schema(context.Background(), resource.SchemaRequest{}, &sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	return tfsdk.State{Schema: sresp.Schema}
}

func stackState(t *testing.T, m swarmStackModel) tfsdk.State {
	t.Helper()
	s := swarmStackSchema(t)
	if diags := s.Set(context.Background(), &m); diags.HasError() {
		t.Fatalf("set state: %v", diags)
	}
	return s
}

func stackPlan(t *testing.T, m swarmStackModel) tfsdk.Plan {
	t.Helper()
	var sresp resource.SchemaResponse
	(&SwarmStackResource{}).Schema(context.Background(), resource.SchemaRequest{}, &sresp)
	p := tfsdk.Plan{Schema: sresp.Schema}
	if diags := p.Set(context.Background(), &m); diags.HasError() {
		t.Fatalf("set plan: %v", diags)
	}
	return p
}

// fullStackModel returns a model with every field populated so tfsdk.Set encodes
// a complete object. Callers override the fields under test.
func fullStackModel() swarmStackModel {
	return swarmStackModel{
		ID:               types.StringValue("web"),
		EnvironmentID:    types.StringValue("env-1"),
		Name:             types.StringValue("web"),
		ComposeContent:   types.StringValue("services:\n  web:\n    image: nginx\n"),
		EnvContent:       types.StringNull(),
		Prune:            types.BoolNull(),
		ResolveImage:     types.StringNull(),
		WithRegistryAuth: types.BoolNull(),
		WorkingDir:       types.StringNull(),
		Namespace:        types.StringValue("web"),
		Services:         types.Int64Value(1),
		CreatedAt:        types.StringValue("2026-01-01T00:00:00Z"),
		UpdatedAt:        types.StringValue("2026-01-01T00:00:00Z"),
	}
}

func newTestClient(t *testing.T, srv *httptest.Server) *sdkclient.Client {
	t.Helper()
	return sdkclient.NewClient(srv.URL, "test-key")
}

// TestSwarmStackRead_PreservesConfiguredEnvWhenSourceEmpty locks in the fix for
// the "missing env" bug: the source endpoint may not echo envContent back, so a
// user-configured env_content must not be wiped to null on refresh.
func TestSwarmStackRead_PreservesConfiguredEnvWhenSourceEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/source"):
			// Server returns no envContent at all.
			w.Write([]byte(`{"success":true,"data":{"name":"web","composeContent":"services:\n  web:\n    image: nginx\n"}}`))
		default:
			w.Write([]byte(`{"success":true,"data":{"name":"web","namespace":"web","services":1,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-02-02T00:00:00Z"}}`))
		}
	}))
	defer srv.Close()

	r := &SwarmStackResource{client: newTestClient(t, srv)}
	prior := fullStackModel()
	prior.EnvContent = types.StringValue("FOO=bar")

	req := resource.ReadRequest{State: stackState(t, prior)}
	resp := &resource.ReadResponse{State: stackState(t, prior)}
	r.Read(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}

	var got swarmStackModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get state: %v", diags)
	}
	if got.EnvContent.ValueString() != "FOO=bar" {
		t.Errorf("env_content was clobbered: got %q, want %q", got.EnvContent.ValueString(), "FOO=bar")
	}
}

// TestSwarmStackRead_RefreshesEnvFromServerWhenConfigured verifies that when the
// user did configure env_content and the server returns a value, refresh adopts
// the server value (drift detection still works).
func TestSwarmStackRead_RefreshesEnvFromServerWhenConfigured(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/source") {
			w.Write([]byte(`{"success":true,"data":{"name":"web","composeContent":"services:\n  web:\n    image: nginx\n","envContent":"FOO=changed"}}`))
			return
		}
		w.Write([]byte(`{"success":true,"data":{"name":"web","namespace":"web","services":1,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`))
	}))
	defer srv.Close()

	r := &SwarmStackResource{client: newTestClient(t, srv)}
	prior := fullStackModel()
	prior.EnvContent = types.StringValue("FOO=bar")

	resp := &resource.ReadResponse{State: stackState(t, prior)}
	r.Read(context.Background(), resource.ReadRequest{State: stackState(t, prior)}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}

	var got swarmStackModel
	resp.State.Get(context.Background(), &got)
	if got.EnvContent.ValueString() != "FOO=changed" {
		t.Errorf("env drift not detected: got %q, want %q", got.EnvContent.ValueString(), "FOO=changed")
	}
}

// TestSwarmStackRead_LeavesEnvNullWhenUnset verifies an unconfigured env_content
// stays null after refresh even if the server returns content.
func TestSwarmStackRead_LeavesEnvNullWhenUnset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/source") {
			w.Write([]byte(`{"success":true,"data":{"name":"web","composeContent":"services:\n  web:\n    image: nginx\n","envContent":"GENERATED=1"}}`))
			return
		}
		w.Write([]byte(`{"success":true,"data":{"name":"web","namespace":"web","services":1,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`))
	}))
	defer srv.Close()

	r := &SwarmStackResource{client: newTestClient(t, srv)}
	prior := fullStackModel() // EnvContent is null

	resp := &resource.ReadResponse{State: stackState(t, prior)}
	r.Read(context.Background(), resource.ReadRequest{State: stackState(t, prior)}, resp)

	var got swarmStackModel
	resp.State.Get(context.Background(), &got)
	if !got.EnvContent.IsNull() {
		t.Errorf("unset env_content adopted a server default: got %q, want null", got.EnvContent.ValueString())
	}
}

// TestSwarmStackUpdate_KeepsPlanContentAndTimestamp locks in two fixes:
//   - compose_content/env_content stay at the configured plan values (no
//     "inconsistent result after apply" when the server normalizes content).
//   - updated_at is left at the prior state value (no plan inconsistency from a
//     server-side timestamp bump combined with UseStateForUnknown).
func TestSwarmStackUpdate_KeepsPlanContentAndTimestamp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/source") {
			// Server echoes a normalized/reformatted body.
			w.Write([]byte(`{"success":true,"data":{"name":"web","composeContent":"NORMALIZED","envContent":"SERVER=normalized"}}`))
			return
		}
		// GET inspect after update, with a bumped timestamp.
		w.Write([]byte(`{"success":true,"data":{"name":"web","namespace":"web","services":2,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-09-09T00:00:00Z"}}`))
	}))
	defer srv.Close()

	r := &SwarmStackResource{client: newTestClient(t, srv)}

	prior := fullStackModel()
	prior.UpdatedAt = types.StringValue("2026-01-01T00:00:00Z")

	plan := fullStackModel()
	plan.ComposeContent = types.StringValue("services:\n  web:\n    image: nginx:1.27\n")
	plan.EnvContent = types.StringValue("FOO=bar")
	// updated_at is computed+UseStateForUnknown: in the plan it equals prior state.
	plan.UpdatedAt = types.StringValue("2026-01-01T00:00:00Z")

	req := resource.UpdateRequest{Plan: stackPlan(t, plan), State: stackState(t, prior)}
	resp := &resource.UpdateResponse{State: stackState(t, prior)}
	r.Update(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", resp.Diagnostics)
	}

	var got swarmStackModel
	resp.State.Get(context.Background(), &got)

	if got.ComposeContent.ValueString() != plan.ComposeContent.ValueString() {
		t.Errorf("compose_content took server value: got %q, want plan value %q", got.ComposeContent.ValueString(), plan.ComposeContent.ValueString())
	}
	if got.EnvContent.ValueString() != "FOO=bar" {
		t.Errorf("env_content took server value: got %q, want %q", got.EnvContent.ValueString(), "FOO=bar")
	}
	if got.UpdatedAt.ValueString() != "2026-01-01T00:00:00Z" {
		t.Errorf("updated_at changed during apply (would be inconsistent with plan): got %q, want %q", got.UpdatedAt.ValueString(), "2026-01-01T00:00:00Z")
	}
	// Computed-only fields that reflect real state are still refreshed.
	if got.Services.ValueInt64() != 2 {
		t.Errorf("services not refreshed: got %d, want 2", got.Services.ValueInt64())
	}
}
