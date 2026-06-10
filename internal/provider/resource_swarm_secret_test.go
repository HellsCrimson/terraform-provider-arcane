package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func secretState(t *testing.T, m swarmSecretModel) tfsdk.State {
	t.Helper()
	var sresp resource.SchemaResponse
	(&SwarmSecretResource{}).Schema(context.Background(), resource.SchemaRequest{}, &sresp)
	s := tfsdk.State{Schema: sresp.Schema}
	if diags := s.Set(context.Background(), &m); diags.HasError() {
		t.Fatalf("set state: %v", diags)
	}
	return s
}

func secretPlan(t *testing.T, m swarmSecretModel) tfsdk.Plan {
	t.Helper()
	var sresp resource.SchemaResponse
	(&SwarmSecretResource{}).Schema(context.Background(), resource.SchemaRequest{}, &sresp)
	p := tfsdk.Plan{Schema: sresp.Schema}
	if diags := p.Set(context.Background(), &m); diags.HasError() {
		t.Fatalf("set plan: %v", diags)
	}
	return p
}

func fullSecretModel() swarmSecretModel {
	return swarmSecretModel{
		ID:            types.StringValue("sec-1"),
		EnvironmentID: types.StringValue("env-1"),
		Name:          types.StringValue("db_password"),
		Data:          types.StringValue("s3cr3t"),
		Labels:        types.MapNull(types.StringType),
		VersionIndex:  types.Int64Value(10),
		CreatedAt:     types.StringValue("2026-01-01T00:00:00Z"),
		UpdatedAt:     types.StringValue("2026-01-01T00:00:00Z"),
	}
}

// TestSwarmSecretRead_PreservesNullLabels locks in the fix that keeps an
// unconfigured (null) optional-only labels attribute null on refresh, even when
// the server returns labels — otherwise the RequiresReplace label diff would
// force a spurious destroy/recreate.
func TestSwarmSecretRead_PreservesNullLabels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"success":true,"data":{"id":"sec-1","spec":{"Name":"db_password","Labels":{"com.docker.stack.namespace":"web"}},"version":{"Index":10},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`))
	}))
	defer srv.Close()

	r := &SwarmSecretResource{client: sdkclient.NewClient(srv.URL, "k")}
	prior := fullSecretModel() // Labels null, Data set

	resp := &resource.ReadResponse{State: secretState(t, prior)}
	r.Read(context.Background(), resource.ReadRequest{State: secretState(t, prior)}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}

	var got swarmSecretModel
	resp.State.Get(context.Background(), &got)
	if !got.Labels.IsNull() {
		t.Errorf("null labels adopted server labels (would force replace): got %v", got.Labels)
	}
	// Plaintext data is write-only on the API and must be preserved from state.
	if got.Data.ValueString() != "s3cr3t" {
		t.Errorf("data not preserved on read: got %q, want %q", got.Data.ValueString(), "s3cr3t")
	}
}

// TestSwarmSecretCreate_EncodesDataBase64 verifies the v2.0.1 wire contract:
// plaintext data is base64-encoded and the spec uses PascalCase field names.
func TestSwarmSecretCreate_EncodesDataBase64(t *testing.T) {
	var gotBody sdkclient.SwarmSecretCreateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Write([]byte(`{"success":true,"data":{"id":"sec-1","spec":{"Name":"db_password"},"version":{"Index":1},"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`))
	}))
	defer srv.Close()

	r := &SwarmSecretResource{client: sdkclient.NewClient(srv.URL, "k")}
	plan := fullSecretModel()
	plan.ID = types.StringNull()
	plan.VersionIndex = types.Int64Null()
	plan.CreatedAt = types.StringNull()
	plan.UpdatedAt = types.StringNull()

	resp := &resource.CreateResponse{State: secretState(t, fullSecretModel())}
	r.Create(context.Background(), resource.CreateRequest{Plan: secretPlan(t, plan)}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", resp.Diagnostics)
	}

	want := base64.StdEncoding.EncodeToString([]byte("s3cr3t"))
	if gotBody.Spec.Data != want {
		t.Errorf("secret data not base64-encoded: got %q, want %q", gotBody.Spec.Data, want)
	}
	if gotBody.Spec.Name != "db_password" {
		t.Errorf("secret spec Name not sent: got %q", gotBody.Spec.Name)
	}
}
