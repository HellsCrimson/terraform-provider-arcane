package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Arcane stores project_name on the sync record and never renames the project
// the sync is bound to, so arcane_gitops_sync renames it itself — which puts it
// under the same "only a stopped project is renamed" rule as arcane_project.
// The fake below is that pair of endpoints: the sync record, which accepts any
// project name, and the project, which does not.

type fakeArcaneGitOpsSync struct {
	project *fakeArcaneProject
	// projectName is the name on the sync record, which the API lets drift from
	// the project's own name.
	projectName string
}

func (f *fakeArcaneGitOpsSync) server(t *testing.T) *httptest.Server {
	t.Helper()

	projectHandler := f.project.handler(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/gitops-syncs/") {
			projectHandler(w, r)
			return
		}

		f.project.mu.Lock()
		defer f.project.mu.Unlock()
		f.project.calls = append(f.project.calls, r.Method+" /gitops-syncs/s1")

		if r.Method == http.MethodPut {
			var body struct {
				ProjectName *string `json:"projectName"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decoding sync update body: %s", err)
			}
			if body.ProjectName != nil {
				f.projectName = *body.ProjectName
			}
		}
		fmt.Fprintf(w, `{"success":true,"data":{"id":"s1","name":"sync","environmentId":"env-1","repositoryId":"repo-1","branch":"main","composePath":"docker-compose.yml","projectName":%q,"projectId":"p1","autoSync":false,"syncInterval":300,"maxSyncBinarySize":0,"maxSyncFiles":0,"maxSyncTotalSize":0,"syncDirectory":false,"targetType":"","enabled":true,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`,
			f.projectName)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// renameGitOpsSyncModel returns a fully populated model: tfsdk only encodes an
// object when every attribute has a value, null included.
func renameGitOpsSyncModel(projectName string) gitOpsSyncModel {
	return gitOpsSyncModel{
		ID:                   types.StringValue("s1"),
		EnvironmentID:        types.StringValue("env-1"),
		Name:                 types.StringValue("sync"),
		RepositoryID:         types.StringValue("repo-1"),
		Branch:               types.StringValue("main"),
		ComposePath:          types.StringValue("docker-compose.yml"),
		ProjectName:          types.StringValue(projectName),
		AutoSync:             types.BoolNull(),
		SyncInterval:         types.Int64Null(),
		MaxSyncBinarySize:    types.Int64Null(),
		MaxSyncFiles:         types.Int64Null(),
		MaxSyncTotalSize:     types.Int64Null(),
		SyncDirectory:        types.BoolNull(),
		TargetType:           types.StringNull(),
		Enabled:              types.BoolValue(true),
		EnvironmentVariables: types.MapNull(types.StringType),
		StartProject:         types.BoolNull(),
		FailIfNameExists:     types.BoolNull(),
		StopBeforeRename:     types.BoolNull(),
		ProjectID:            types.StringValue("p1"),
		LastSyncAt:           types.StringNull(),
		LastSyncCommit:       types.StringNull(),
		LastSyncStatus:       types.StringNull(),
		LastSyncError:        types.StringNull(),
		CreatedAt:            types.StringValue("2026-01-01T00:00:00Z"),
		UpdatedAt:            types.StringValue("2026-01-01T00:00:00Z"),
	}
}

func gitOpsSyncSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()

	var sresp resource.SchemaResponse
	(&GitOpsSyncResource{}).Schema(context.Background(), resource.SchemaRequest{}, &sresp)
	if sresp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", sresp.Diagnostics)
	}
	return sresp
}

func gitOpsSyncState(t *testing.T, m gitOpsSyncModel) tfsdk.State {
	t.Helper()

	s := tfsdk.State{Schema: gitOpsSyncSchema(t).Schema}
	if diags := s.Set(context.Background(), &m); diags.HasError() {
		t.Fatalf("set state: %v", diags)
	}
	return s
}

func gitOpsSyncPlan(t *testing.T, m gitOpsSyncModel) tfsdk.Plan {
	t.Helper()

	p := tfsdk.Plan{Schema: gitOpsSyncSchema(t).Schema}
	if diags := p.Set(context.Background(), &m); diags.HasError() {
		t.Fatalf("set plan: %v", diags)
	}
	return p
}

// modifyGitOpsSyncPlan runs ModifyPlan for a plan against the given state, and
// returns the diagnostics it produced.
func modifyGitOpsSyncPlan(t *testing.T, fake *fakeArcaneGitOpsSync, plan, state gitOpsSyncModel) *resource.ModifyPlanResponse {
	t.Helper()

	srv := fake.server(t)
	r := &GitOpsSyncResource{client: newTestClient(t, srv)}

	req := resource.ModifyPlanRequest{
		Plan:  gitOpsSyncPlan(t, plan),
		State: gitOpsSyncState(t, state),
	}
	resp := &resource.ModifyPlanResponse{Plan: gitOpsSyncPlan(t, plan)}
	r.ModifyPlan(context.Background(), req, resp)
	return resp
}

// TestGitOpsSyncModifyPlan_ProjectRenameWhileRunningFailsPlan is the issue #23
// check for the GitOps-managed project: changing project_name renames a real
// project, so a running one has to fail the plan rather than the apply.
func TestGitOpsSyncModifyPlan_ProjectRenameWhileRunningFailsPlan(t *testing.T) {
	fake := &fakeArcaneGitOpsSync{project: &fakeArcaneProject{name: "old", running: true}, projectName: "old"}

	resp := modifyGitOpsSyncPlan(t, fake, renameGitOpsSyncModel("new"), renameGitOpsSyncModel("old"))

	if !hasRenameError(resp) {
		t.Fatalf("expected a %q error, got: %v", renameRequiresStopSummary, resp.Diagnostics)
	}
	for _, d := range resp.Diagnostics.Errors() {
		if !strings.Contains(d.Detail(), "stop_before_rename = true") {
			t.Errorf("error does not point at the way out: %s", d.Detail())
		}
	}
}

// TestGitOpsSyncModifyPlan_ProjectRenameAllowed covers the plans that must not
// fail: the opt-in, a project that is already stopped, and a sync that has no
// project yet (the next sync creates it under the new name).
func TestGitOpsSyncModifyPlan_ProjectRenameAllowed(t *testing.T) {
	stopBeforeRename := renameGitOpsSyncModel("new")
	stopBeforeRename.StopBeforeRename = types.BoolValue(true)

	// The first sync has not created the project yet, so there is nothing to
	// rename: it is created under the new name.
	noProjectYet := renameGitOpsSyncModel("old")
	noProjectYet.ProjectID = types.StringNull()

	for _, tc := range []struct {
		name    string
		running bool
		plan    gitOpsSyncModel
		state   gitOpsSyncModel
	}{
		{"stop_before_rename", true, stopBeforeRename, renameGitOpsSyncModel("old")},
		{"project already stopped", false, renameGitOpsSyncModel("new"), renameGitOpsSyncModel("old")},
		{"no project bound yet", true, renameGitOpsSyncModel("new"), noProjectYet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeArcaneGitOpsSync{project: &fakeArcaneProject{name: "old", running: tc.running}, projectName: "old"}

			resp := modifyGitOpsSyncPlan(t, fake, tc.plan, tc.state)

			if resp.Diagnostics.HasError() {
				t.Fatalf("plan failed: %v", resp.Diagnostics)
			}
		})
	}
}

// TestGitOpsSyncModifyPlan_NoRenameDoesNotCallAPI keeps the check off the hot
// path: a plan that leaves project_name alone must not talk to Arcane at all.
func TestGitOpsSyncModifyPlan_NoRenameDoesNotCallAPI(t *testing.T) {
	fake := &fakeArcaneGitOpsSync{project: &fakeArcaneProject{name: "old", running: true}, projectName: "old"}

	plan := renameGitOpsSyncModel("old")
	plan.Branch = types.StringValue("release")

	resp := modifyGitOpsSyncPlan(t, fake, plan, renameGitOpsSyncModel("old"))

	if resp.Diagnostics.HasError() {
		t.Fatalf("plan failed: %v", resp.Diagnostics)
	}
	if calls := fake.project.recorded(); len(calls) != 0 {
		t.Errorf("plan called the API for a change that renames nothing: %v", calls)
	}
}

func updateGitOpsSync(t *testing.T, fake *fakeArcaneGitOpsSync, plan gitOpsSyncModel) *resource.UpdateResponse {
	t.Helper()

	srv := fake.server(t)
	r := &GitOpsSyncResource{client: newTestClient(t, srv)}

	state := renameGitOpsSyncModel("old")
	req := resource.UpdateRequest{
		Plan:  gitOpsSyncPlan(t, plan),
		State: gitOpsSyncState(t, state),
	}
	resp := &resource.UpdateResponse{State: gitOpsSyncState(t, state)}
	r.Update(context.Background(), req, resp)
	return resp
}

// TestGitOpsSyncUpdate_StopBeforeRenameRenamesProject pins down the apply half:
// the sync record alone never renames anything, so the provider has to stop the
// bound project, rename it and start it again.
func TestGitOpsSyncUpdate_StopBeforeRenameRenamesProject(t *testing.T) {
	fake := &fakeArcaneGitOpsSync{project: &fakeArcaneProject{name: "old", running: true}, projectName: "old"}

	plan := renameGitOpsSyncModel("new")
	plan.StopBeforeRename = types.BoolValue(true)

	resp := updateGitOpsSync(t, fake, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", resp.Diagnostics)
	}

	want := []string{"PUT /gitops-syncs/s1", "GET ", "POST /down", "PUT ", "POST /up"}
	if got := fake.project.recorded(); !equalStrings(got, want) {
		t.Errorf("unexpected API calls:\n got: %v\nwant: %v", got, want)
	}
	if fake.project.name != "new" {
		t.Errorf("project name after rename: got %q, want %q", fake.project.name, "new")
	}
	if !fake.project.running {
		t.Error("project was left stopped after the rename")
	}

	var got gitOpsSyncModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get state: %v", diags)
	}
	if got.ProjectName.ValueString() != "new" {
		t.Errorf("project_name in state: got %q, want %q", got.ProjectName.ValueString(), "new")
	}
	if got.StopBeforeRename.ValueBool() != true {
		t.Errorf("stop_before_rename in state: got %v, want true", got.StopBeforeRename)
	}
}

// TestGitOpsSyncUpdate_RenameWhileRunningFails is the safety net behind the
// plan check: state that was never refreshed can carry a stale status into the
// apply, and the provider must not stop a project it was not told to stop.
func TestGitOpsSyncUpdate_RenameWhileRunningFails(t *testing.T) {
	fake := &fakeArcaneGitOpsSync{project: &fakeArcaneProject{name: "old", running: true}, projectName: "old"}

	resp := updateGitOpsSync(t, fake, renameGitOpsSyncModel("new"))

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected the update to fail for a running project")
	}
	if fake.project.name != "old" {
		t.Errorf("project was renamed anyway: %q", fake.project.name)
	}
	if !fake.project.running {
		t.Error("project was stopped without stop_before_rename")
	}
}

// TestGitOpsSyncUpdate_AlreadyStoppedRenamesInPlace covers the project that is
// stopped to begin with: it is renamed without a stop, and stays stopped.
func TestGitOpsSyncUpdate_AlreadyStoppedRenamesInPlace(t *testing.T) {
	fake := &fakeArcaneGitOpsSync{project: &fakeArcaneProject{name: "old", running: false}, projectName: "old"}

	resp := updateGitOpsSync(t, fake, renameGitOpsSyncModel("new"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", resp.Diagnostics)
	}

	want := []string{"PUT /gitops-syncs/s1", "GET ", "PUT "}
	if got := fake.project.recorded(); !equalStrings(got, want) {
		t.Errorf("unexpected API calls:\n got: %v\nwant: %v", got, want)
	}
	if fake.project.name != "new" {
		t.Errorf("project name after rename: got %q, want %q", fake.project.name, "new")
	}
	if fake.project.running {
		t.Error("project was started by a rename that did not stop it")
	}
}
