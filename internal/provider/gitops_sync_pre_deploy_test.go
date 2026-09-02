package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The pre-deploy lifecycle hook fields share one drift trap: the API answers
// with server-side defaults for preDeployNetworkMode ("none") and
// preDeployTimeoutSec (60) even when the practitioner never configured them,
// and writing those defaults into state where the configuration is null makes
// Terraform fail with "Provider produced inconsistent result after apply".
// The tests below pin the same handling the resource already uses for
// sync_interval, max_sync_* and target_type: server values are only taken when
// the plan (or prior state, on Read) carries a value.

// fakeGitOpsSyncPreDeploy is a minimal Arcane fake for the sync endpoints. It
// records the JSON body of the last PUT and echoes the pre-deploy fields it
// received, on top of the server defaults the real API always reports.
type fakeGitOpsSyncPreDeploy struct {
	lastPut map[string]any
	// maxTimeoutSec, when non-zero, makes the fake clamp preDeployTimeoutSec
	// the way a server bounded by lifecycle_max_timeout_sec does: the request
	// is accepted, but the response reports the capped value.
	maxTimeoutSec int64
}

func (f *fakeGitOpsSyncPreDeploy) server(t *testing.T) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		extra := ""
		if r.Method == http.MethodPut {
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("reading sync update body: %s", err)
			}
			f.lastPut = map[string]any{}
			if err := json.Unmarshal(raw, &f.lastPut); err != nil {
				t.Errorf("decoding sync update body: %s", err)
			}
		}
		networkMode, timeoutSec := "none", int64(60)
		if f.lastPut != nil {
			for _, k := range []string{"preDeployScriptPath", "preDeployRunnerImage", "preDeployEnv", "preDeployExtraMounts"} {
				if v, ok := f.lastPut[k].(string); ok && v != "" {
					extra += fmt.Sprintf("%q:%q,", k, v)
				}
			}
			if v, ok := f.lastPut["preDeployNetworkMode"].(string); ok && v != "" {
				networkMode = v
			}
			if v, ok := f.lastPut["preDeployTimeoutSec"].(float64); ok {
				timeoutSec = int64(v)
			}
			if f.maxTimeoutSec > 0 && timeoutSec > f.maxTimeoutSec {
				timeoutSec = f.maxTimeoutSec
			}
		}
		fmt.Fprintf(w, `{"success":true,"data":{"id":"s1","name":"sync","environmentId":"env-1","repositoryId":"repo-1","branch":"main","composePath":"docker-compose.yml","projectName":"old","projectId":"p1","autoSync":false,"syncInterval":300,"maxSyncBinarySize":0,"maxSyncFiles":0,"maxSyncTotalSize":0,"syncDirectory":false,"targetType":"","enabled":true,%s"preDeployNetworkMode":%q,"preDeployTimeoutSec":%d,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`,
			extra, networkMode, timeoutSec)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func updateGitOpsSyncPreDeploy(t *testing.T, fake *fakeGitOpsSyncPreDeploy, plan, state gitOpsSyncModel) *resource.UpdateResponse {
	t.Helper()

	srv := fake.server(t)
	r := &GitOpsSyncResource{client: newTestClient(t, srv)}

	req := resource.UpdateRequest{
		Plan:  gitOpsSyncPlan(t, plan),
		State: gitOpsSyncState(t, state),
	}
	resp := &resource.UpdateResponse{State: gitOpsSyncState(t, state)}
	r.Update(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", resp.Diagnostics)
	}
	return resp
}

// TestGitOpsSyncUpdate_PreDeployFieldsSent checks the whole configured path:
// every pre-deploy attribute set in the plan reaches the request body and comes
// back into state.
func TestGitOpsSyncUpdate_PreDeployFieldsSent(t *testing.T) {
	plan := renameGitOpsSyncModel("old")
	plan.PreDeployScriptPath = types.StringValue("pre-deploy.sh")
	plan.PreDeployRunnerImage = types.StringValue("ghcr.io/getsops/sops:v3.11.0")
	plan.PreDeployEnv = types.StringValue("SOPS_AGE_KEY_FILE=/run/secrets/age.key")
	plan.PreDeployExtraMounts = types.StringValue("/opt/arcane/secrets/age.key:/run/secrets/age.key:ro")
	plan.PreDeployTimeoutSec = types.Int64Value(120)
	plan.PreDeployNetworkMode = types.StringValue("bridge")

	fake := &fakeGitOpsSyncPreDeploy{}
	resp := updateGitOpsSyncPreDeploy(t, fake, plan, renameGitOpsSyncModel("old"))

	wantBody := map[string]any{
		"preDeployScriptPath":  "pre-deploy.sh",
		"preDeployRunnerImage": "ghcr.io/getsops/sops:v3.11.0",
		"preDeployEnv":         "SOPS_AGE_KEY_FILE=/run/secrets/age.key",
		"preDeployExtraMounts": "/opt/arcane/secrets/age.key:/run/secrets/age.key:ro",
		"preDeployTimeoutSec":  float64(120),
		"preDeployNetworkMode": "bridge",
	}
	for k, want := range wantBody {
		if got := fake.lastPut[k]; got != want {
			t.Errorf("update body %s: got %v, want %v", k, got, want)
		}
	}

	var got gitOpsSyncModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get state: %v", diags)
	}
	if got.PreDeployScriptPath.ValueString() != "pre-deploy.sh" {
		t.Errorf("pre_deploy_script_path in state: got %q", got.PreDeployScriptPath.ValueString())
	}
	if got.PreDeployTimeoutSec.ValueInt64() != 120 {
		t.Errorf("pre_deploy_timeout_sec in state: got %d", got.PreDeployTimeoutSec.ValueInt64())
	}
	if got.PreDeployNetworkMode.ValueString() != "bridge" {
		t.Errorf("pre_deploy_network_mode in state: got %q", got.PreDeployNetworkMode.ValueString())
	}
}

// TestGitOpsSyncUpdate_PreDeployServerDefaultsStayNull is the drift check: a
// plan that never mentions the hook must neither send pre-deploy fields nor let
// the server defaults ("none", 60) leak into state where the plan is null.
func TestGitOpsSyncUpdate_PreDeployServerDefaultsStayNull(t *testing.T) {
	plan := renameGitOpsSyncModel("old")
	plan.Branch = types.StringValue("release")

	fake := &fakeGitOpsSyncPreDeploy{}
	resp := updateGitOpsSyncPreDeploy(t, fake, plan, renameGitOpsSyncModel("old"))

	for _, k := range []string{"preDeployScriptPath", "preDeployRunnerImage", "preDeployEnv", "preDeployExtraMounts", "preDeployTimeoutSec", "preDeployNetworkMode"} {
		if v, ok := fake.lastPut[k]; ok {
			t.Errorf("update body carries %s = %v for a plan that never set it", k, v)
		}
	}

	var got gitOpsSyncModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get state: %v", diags)
	}
	if !got.PreDeployNetworkMode.IsNull() {
		t.Errorf("pre_deploy_network_mode picked up the server default: %v", got.PreDeployNetworkMode)
	}
	if !got.PreDeployTimeoutSec.IsNull() {
		t.Errorf("pre_deploy_timeout_sec picked up the server default: %v", got.PreDeployTimeoutSec)
	}
	if !got.PreDeployScriptPath.IsNull() || !got.PreDeployRunnerImage.IsNull() || !got.PreDeployEnv.IsNull() || !got.PreDeployExtraMounts.IsNull() {
		t.Error("a pre-deploy attribute the plan never set became non-null")
	}
}

// TestGitOpsSyncUpdate_PreDeployRemovalClears covers taking the hook out of the
// configuration: Arcane keeps a hook it is not told about, so the provider has
// to send empty strings to clear it (the API's documented reset semantics).
func TestGitOpsSyncUpdate_PreDeployRemovalClears(t *testing.T) {
	state := renameGitOpsSyncModel("old")
	state.PreDeployScriptPath = types.StringValue("pre-deploy.sh")
	state.PreDeployRunnerImage = types.StringValue("alpine:3")
	state.PreDeployEnv = types.StringValue("A=b")
	state.PreDeployExtraMounts = types.StringValue("/a:/b:ro")
	state.PreDeployNetworkMode = types.StringValue("bridge")

	fake := &fakeGitOpsSyncPreDeploy{}
	resp := updateGitOpsSyncPreDeploy(t, fake, renameGitOpsSyncModel("old"), state)

	for _, k := range []string{"preDeployScriptPath", "preDeployRunnerImage", "preDeployEnv", "preDeployExtraMounts", "preDeployNetworkMode"} {
		v, ok := fake.lastPut[k]
		if !ok {
			t.Errorf("update body omits %s, which leaves the hook configured on the server", k)
			continue
		}
		if v != "" {
			t.Errorf("update body %s: got %v, want \"\" (clear)", k, v)
		}
	}

	var got gitOpsSyncModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get state: %v", diags)
	}
	if !got.PreDeployScriptPath.IsNull() || !got.PreDeployNetworkMode.IsNull() {
		t.Error("cleared pre-deploy attributes did not return to null in state")
	}
}

// TestGitOpsSyncRead_PreDeployServerDefaultsStayNull is the refresh half of the
// drift check: a state whose pre-deploy attributes are null must stay null even
// though the GET response reports the server defaults.
func TestGitOpsSyncRead_PreDeployServerDefaultsStayNull(t *testing.T) {
	fake := &fakeGitOpsSyncPreDeploy{}
	srv := fake.server(t)
	r := &GitOpsSyncResource{client: newTestClient(t, srv)}

	state := renameGitOpsSyncModel("old")
	req := resource.ReadRequest{State: gitOpsSyncState(t, state)}
	resp := &resource.ReadResponse{State: gitOpsSyncState(t, state)}
	r.Read(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read failed: %v", resp.Diagnostics)
	}

	var got gitOpsSyncModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get state: %v", diags)
	}
	if !got.PreDeployNetworkMode.IsNull() {
		t.Errorf("pre_deploy_network_mode picked up the server default on refresh: %v", got.PreDeployNetworkMode)
	}
	if !got.PreDeployTimeoutSec.IsNull() {
		t.Errorf("pre_deploy_timeout_sec picked up the server default on refresh: %v", got.PreDeployTimeoutSec)
	}
}

// TestGitOpsSyncRead_PreDeployConfiguredValuesRefresh is the counterpart: when
// the practitioner configured the hook, a refresh reflects what the server has,
// so out-of-band changes surface as drift.
func TestGitOpsSyncRead_PreDeployConfiguredValuesRefresh(t *testing.T) {
	fake := &fakeGitOpsSyncPreDeploy{lastPut: map[string]any{
		"preDeployScriptPath":  "pre-deploy.sh",
		"preDeployRunnerImage": "alpine:3",
		"preDeployNetworkMode": "host",
		"preDeployTimeoutSec":  float64(300),
	}}
	srv := fake.server(t)
	r := &GitOpsSyncResource{client: newTestClient(t, srv)}

	state := renameGitOpsSyncModel("old")
	state.PreDeployScriptPath = types.StringValue("old.sh")
	state.PreDeployRunnerImage = types.StringValue("alpine:2")
	state.PreDeployNetworkMode = types.StringValue("none")
	state.PreDeployTimeoutSec = types.Int64Value(60)

	req := resource.ReadRequest{State: gitOpsSyncState(t, state)}
	resp := &resource.ReadResponse{State: gitOpsSyncState(t, state)}
	r.Read(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read failed: %v", resp.Diagnostics)
	}

	var got gitOpsSyncModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get state: %v", diags)
	}
	if got.PreDeployScriptPath.ValueString() != "pre-deploy.sh" {
		t.Errorf("pre_deploy_script_path after refresh: got %q, want %q", got.PreDeployScriptPath.ValueString(), "pre-deploy.sh")
	}
	if got.PreDeployRunnerImage.ValueString() != "alpine:3" {
		t.Errorf("pre_deploy_runner_image after refresh: got %q, want %q", got.PreDeployRunnerImage.ValueString(), "alpine:3")
	}
	if got.PreDeployNetworkMode.ValueString() != "host" {
		t.Errorf("pre_deploy_network_mode after refresh: got %q, want %q", got.PreDeployNetworkMode.ValueString(), "host")
	}
	if got.PreDeployTimeoutSec.ValueInt64() != 300 {
		t.Errorf("pre_deploy_timeout_sec after refresh: got %d, want 300", got.PreDeployTimeoutSec.ValueInt64())
	}
}

// TestGitOpsSyncUpdate_PreDeployTimeoutRemovalClears is the timeout half of
// TestGitOpsSyncUpdate_PreDeployRemovalClears. Every other pre-deploy attribute
// sends its zero value to reset the server when it leaves the configuration;
// pre_deploy_timeout_sec has to do the same, or the server silently keeps the
// old timeout while state says null and no later plan ever shows the gap
// (Read skips attributes that are null in state).
func TestGitOpsSyncUpdate_PreDeployTimeoutRemovalClears(t *testing.T) {
	state := renameGitOpsSyncModel("old")
	state.PreDeployScriptPath = types.StringValue("pre-deploy.sh")
	state.PreDeployRunnerImage = types.StringValue("alpine:3")
	state.PreDeployTimeoutSec = types.Int64Value(300)

	fake := &fakeGitOpsSyncPreDeploy{}
	resp := updateGitOpsSyncPreDeploy(t, fake, renameGitOpsSyncModel("old"), state)

	// The body field is *int64 with omitempty, so a pointer to 0 is still
	// encoded: 0 is the reset signal, matching "" for the string attributes.
	v, ok := fake.lastPut["preDeployTimeoutSec"]
	if !ok {
		t.Error("update body omits preDeployTimeoutSec, which leaves the old timeout on the server")
	} else if v != float64(0) {
		t.Errorf("update body preDeployTimeoutSec: got %v, want 0 (reset to the server default)", v)
	}

	var got gitOpsSyncModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get state: %v", diags)
	}
	if !got.PreDeployTimeoutSec.IsNull() {
		t.Errorf("pre_deploy_timeout_sec after removal: got %v, want null", got.PreDeployTimeoutSec)
	}
}

// TestGitOpsSyncUpdate_PreDeployUnknownIsNotRemoval separates "unknown" from
// "removed". The clear branches hang off an `else` on a guard that rejects both
// null and unknown, so an attribute whose value is not yet known — e.g.
// pre_deploy_runner_image = arcane_something.x.image — takes the reset path and
// wipes a hook the practitioner still wants. Unknown means "no instruction
// yet", never "clear it".
func TestGitOpsSyncUpdate_PreDeployUnknownIsNotRemoval(t *testing.T) {
	state := renameGitOpsSyncModel("old")
	state.PreDeployScriptPath = types.StringValue("pre-deploy.sh")
	state.PreDeployRunnerImage = types.StringValue("alpine:3")
	state.PreDeployEnv = types.StringValue("A=b")
	state.PreDeployNetworkMode = types.StringValue("bridge")

	plan := state
	plan.PreDeployRunnerImage = types.StringUnknown()

	fake := &fakeGitOpsSyncPreDeploy{}
	resp := updateGitOpsSyncPreDeploy(t, fake, plan, state)

	if v, ok := fake.lastPut["preDeployRunnerImage"]; ok && v == "" {
		t.Error("update body clears preDeployRunnerImage for an unknown plan value; the configured hook is destroyed")
	}

	var got gitOpsSyncModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get state: %v", diags)
	}
	// An unknown left in state fails the apply with "Provider returned invalid
	// result object after apply", after the destructive write already landed.
	if got.PreDeployRunnerImage.IsUnknown() {
		t.Error("pre_deploy_runner_image is still unknown in the post-apply state")
	}
}

// TestGitOpsSyncUpdate_PreDeployTimeoutKeepsPlanValue covers a server that
// clamps: pre_deploy_timeout_sec is Optional and not Computed, so Terraform
// requires the post-apply state to equal the configuration exactly. Writing the
// clamped response value instead aborts the apply with "Provider produced
// inconsistent result after apply"; keeping the plan value lets the cap show up
// as ordinary drift on the next refresh instead.
func TestGitOpsSyncUpdate_PreDeployTimeoutKeepsPlanValue(t *testing.T) {
	plan := renameGitOpsSyncModel("old")
	plan.PreDeployScriptPath = types.StringValue("pre-deploy.sh")
	plan.PreDeployRunnerImage = types.StringValue("alpine:3")
	plan.PreDeployTimeoutSec = types.Int64Value(600)

	fake := &fakeGitOpsSyncPreDeploy{maxTimeoutSec: 300}
	resp := updateGitOpsSyncPreDeploy(t, fake, plan, renameGitOpsSyncModel("old"))

	if got := fake.lastPut["preDeployTimeoutSec"]; got != float64(600) {
		t.Errorf("update body preDeployTimeoutSec: got %v, want 600", got)
	}

	var got gitOpsSyncModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("get state: %v", diags)
	}
	if got.PreDeployTimeoutSec.ValueInt64() != 600 {
		t.Errorf("pre_deploy_timeout_sec in state: got %d, want 600 (the planned value, not the server's clamp)", got.PreDeployTimeoutSec.ValueInt64())
	}
}
