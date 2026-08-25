package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fakeArcaneProject stands in for Arcane's project endpoints and enforces the
// one rule these tests are about: a project is only renamed while it is
// stopped, and the update call is rejected the way Arcane rejects it otherwise.
type fakeArcaneProject struct {
	mu      sync.Mutex
	name    string
	running bool
	calls   []string
}

func (f *fakeArcaneProject) status() string {
	if f.running {
		return "running"
	}
	return projectStatusStopped
}

func (f *fakeArcaneProject) writeDetails(w http.ResponseWriter) {
	runningCount := 0
	if f.running {
		runningCount = 1
	}
	fmt.Fprintf(w, `{"success":true,"data":{"id":"p1","name":%q,"path":"/data/projects/%s","status":%q,"serviceCount":1,"runningCount":%d,"isArchived":false,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`,
		f.name, f.name, f.status(), runningCount)
}

func (f *fakeArcaneProject) server(t *testing.T) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(f.handler(t))
	t.Cleanup(srv.Close)
	return srv
}

// handler serves the project endpoints, so a fake that also serves other
// endpoints (see fakeArcaneGitOpsSync) can delegate to it.
func (f *fakeArcaneProject) handler(t *testing.T) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, r.Method+" "+strings.TrimPrefix(r.URL.Path, "/environments/env-1/projects/p1"))

		switch {
		case strings.HasSuffix(r.URL.Path, "/down"):
			f.running = false
			fmt.Fprint(w, `{"success":true}`)
		case strings.HasSuffix(r.URL.Path, "/up"):
			f.running = true
			fmt.Fprint(w, `{"success":true}`)
		case r.Method == http.MethodPut:
			var body struct {
				Name *string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decoding update body: %s", err)
			}
			if body.Name != nil && *body.Name != f.name && f.running {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, `{"title":"Bad Request","status":400,"detail":"Failed to update project: project must be stopped before renaming (current status: running)"}`)
				return
			}
			if body.Name != nil {
				f.name = *body.Name
			}
			f.writeDetails(w)
		default:
			f.writeDetails(w)
		}
	}
}

func (f *fakeArcaneProject) recorded() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

// renameProjectModel returns a fully populated model: tfsdk only encodes an
// object when every attribute has a value, null included.
func renameProjectModel(name string, running bool) projectModel {
	return projectModel{
		ID:               types.StringValue("p1"),
		EnvironmentID:    types.StringValue("env-1"),
		Name:             types.StringValue(name),
		Compose:          types.StringValue("services:\n  app:\n    image: alpine\n"),
		Env:              types.StringNull(),
		Running:          types.BoolValue(running),
		Archived:         types.BoolValue(false),
		RedeployOnUpdate: types.BoolValue(false),
		RedeployTrigger:  types.StringValue(redeployTriggerNever),
		LastRedeploy:     types.StringNull(),
		PullOnUpdate:     types.BoolValue(false),
		RemoveOrphans:    types.BoolNull(),
		FailIfNameExists: types.BoolNull(),
		StopBeforeRename: types.BoolNull(),
		Path:             types.StringValue("/data/projects/" + name),
		Status:           types.StringValue("running"),
		ServiceCount:     types.Int64Value(1),
		RunningCount:     types.Int64Value(1),
		CreatedAt:        types.StringValue("2026-01-01T00:00:00Z"),
		UpdatedAt:        types.StringValue("2026-01-01T00:00:00Z"),
		ArchivedAt:       types.StringNull(),
		IsDiscovered:     types.BoolValue(false),
		RedeployDisabled: types.BoolValue(false),
		RemoveFiles:      types.BoolNull(),
		RemoveVolumes:    types.BoolNull(),
	}
}

func projectSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()

	var sresp resource.SchemaResponse
	(&ProjectResource{}).Schema(context.Background(), resource.SchemaRequest{}, &sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	return sresp
}

func projectState(t *testing.T, m projectModel) tfsdk.State {
	t.Helper()

	s := tfsdk.State{Schema: projectSchema(t).Schema}
	if diags := s.Set(context.Background(), &m); diags.HasError() {
		t.Fatalf("set state: %v", diags)
	}
	return s
}

func projectPlan(t *testing.T, m projectModel) tfsdk.Plan {
	t.Helper()

	p := tfsdk.Plan{Schema: projectSchema(t).Schema}
	if diags := p.Set(context.Background(), &m); diags.HasError() {
		t.Fatalf("set plan: %v", diags)
	}
	return p
}

// projectConfig borrows the raw object from a plan: tfsdk.Config has no Set,
// and the configuration only has to carry the attributes the plan and update
// paths read back from it (the redeploy ones).
func projectConfig(t *testing.T, m projectModel) tfsdk.Config {
	t.Helper()

	p := projectPlan(t, m)
	return tfsdk.Config{Schema: p.Schema, Raw: p.Raw}
}

// modifyProjectPlan runs ModifyPlan for a rename from "old" to "new" against
// the given fake, and returns the diagnostics it produced.
func modifyProjectPlan(t *testing.T, fake *fakeArcaneProject, plan projectModel) *resource.ModifyPlanResponse {
	t.Helper()

	srv := fake.server(t)
	r := &ProjectResource{client: newTestClient(t, srv)}

	state := renameProjectModel("old", true)
	req := resource.ModifyPlanRequest{
		Plan:   projectPlan(t, plan),
		State:  projectState(t, state),
		Config: projectConfig(t, plan),
	}
	resp := &resource.ModifyPlanResponse{Plan: projectPlan(t, plan)}
	r.ModifyPlan(context.Background(), req, resp)
	return resp
}

func hasRenameError(resp *resource.ModifyPlanResponse) bool {
	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == renameRequiresStopSummary {
			return true
		}
	}
	return false
}

// TestProjectModifyPlan_RenameWhileRunningFailsPlan is the unit-level guard for
// issue #23: a rename of a running project has to be rejected while planning,
// because the apply cannot possibly succeed.
func TestProjectModifyPlan_RenameWhileRunningFailsPlan(t *testing.T) {
	fake := &fakeArcaneProject{name: "old", running: true}

	resp := modifyProjectPlan(t, fake, renameProjectModel("new", true))

	if !hasRenameError(resp) {
		t.Fatalf("expected a %q error, got: %v", renameRequiresStopSummary, resp.Diagnostics)
	}
}

// TestProjectModifyPlan_RenameAllowedWhenPlanStopsProject covers the two ways a
// plan satisfies the requirement by itself, plus the case where the project is
// already stopped. None of them may fail the plan.
func TestProjectModifyPlan_RenameAllowedWhenPlanStopsProject(t *testing.T) {
	stopBeforeRename := renameProjectModel("new", true)
	stopBeforeRename.StopBeforeRename = types.BoolValue(true)

	// Lifecycle unmanaged, but archiving brings the project down regardless.
	archived := renameProjectModel("new", true)
	archived.Running = types.BoolNull()
	archived.Archived = types.BoolValue(true)

	for _, tc := range []struct {
		name    string
		running bool
		plan    projectModel
	}{
		{"stop_before_rename", true, stopBeforeRename},
		{"plan stops the project", true, renameProjectModel("new", false)},
		{"plan archives the project", true, archived},
		{"project already stopped", false, renameProjectModel("new", true)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeArcaneProject{name: "old", running: tc.running}

			resp := modifyProjectPlan(t, fake, tc.plan)

			if resp.Diagnostics.HasError() {
				t.Fatalf("plan failed: %v", resp.Diagnostics)
			}
		})
	}
}

// TestProjectModifyPlan_NoRenameDoesNotCallAPI keeps the check off the hot
// path: a plan that leaves the name alone must not talk to Arcane at all.
func TestProjectModifyPlan_NoRenameDoesNotCallAPI(t *testing.T) {
	fake := &fakeArcaneProject{name: "old", running: true}

	plan := renameProjectModel("old", true)
	plan.Compose = types.StringValue("services:\n  app:\n    image: alpine:3\n")

	resp := modifyProjectPlan(t, fake, plan)

	if resp.Diagnostics.HasError() {
		t.Fatalf("plan failed: %v", resp.Diagnostics)
	}
	if calls := fake.recorded(); len(calls) != 0 {
		t.Errorf("plan called the API for a change that renames nothing: %v", calls)
	}
}

// TestProjectUpdate_StopBeforeRenameStopsAndStarts pins down the apply half:
// with stop_before_rename the provider brings the project down, renames it and
// starts it again — in that order, or the fake API rejects the rename the way
// Arcane does.
func TestProjectUpdate_StopBeforeRenameStopsAndStarts(t *testing.T) {
	fake := &fakeArcaneProject{name: "old", running: true}
	srv := fake.server(t)
	r := &ProjectResource{client: newTestClient(t, srv)}

	plan := renameProjectModel("new", true)
	plan.StopBeforeRename = types.BoolValue(true)
	state := renameProjectModel("old", true)

	req := resource.UpdateRequest{
		Plan:   projectPlan(t, plan),
		State:  projectState(t, state),
		Config: projectConfig(t, plan),
	}
	resp := &resource.UpdateResponse{State: projectState(t, state)}
	r.Update(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", resp.Diagnostics)
	}

	want := []string{"GET ", "POST /down", "PUT ", "POST /up", "GET "}
	if got := fake.recorded(); !equalStrings(got, want) {
		t.Errorf("unexpected API calls:\n got: %v\nwant: %v", got, want)
	}

	var got projectModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get state: %v", diags)
	}
	if got.Name.ValueString() != "new" {
		t.Errorf("name after rename: got %q, want %q", got.Name.ValueString(), "new")
	}
	if got.Status.ValueString() != "running" {
		t.Errorf("status after rename: got %q, want %q", got.Status.ValueString(), "running")
	}
}

// TestProjectUpdate_RenameStopsBeforeUpdateWhenPlanStops covers the other
// order-sensitive path: the plan stops the project anyway, so the down has to
// happen before the update call rather than after it.
func TestProjectUpdate_RenameStopsBeforeUpdateWhenPlanStops(t *testing.T) {
	fake := &fakeArcaneProject{name: "old", running: true}
	srv := fake.server(t)
	r := &ProjectResource{client: newTestClient(t, srv)}

	plan := renameProjectModel("new", false)
	state := renameProjectModel("old", true)

	req := resource.UpdateRequest{
		Plan:   projectPlan(t, plan),
		State:  projectState(t, state),
		Config: projectConfig(t, plan),
	}
	resp := &resource.UpdateResponse{State: projectState(t, state)}
	r.Update(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", resp.Diagnostics)
	}

	want := []string{"GET ", "POST /down", "PUT "}
	if got := fake.recorded(); !equalStrings(got, want) {
		t.Errorf("unexpected API calls:\n got: %v\nwant: %v", got, want)
	}

	var got projectModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get state: %v", diags)
	}
	if got.Name.ValueString() != "new" {
		t.Errorf("name after rename: got %q, want %q", got.Name.ValueString(), "new")
	}
	if got.Status.ValueString() != projectStatusStopped {
		t.Errorf("status after rename: got %q, want %q", got.Status.ValueString(), projectStatusStopped)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
