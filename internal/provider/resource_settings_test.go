package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildSettingsMapLifecycleFields locks in the wire mapping of the global
// lifecycle hook settings: each snake_case attribute lands under the exact
// camelCase key the settings API expects, and unset attributes stay out of the
// update map entirely (the API keeps whatever is configured for absent keys).
func TestBuildSettingsMapLifecycleFields(t *testing.T) {
	empty := buildSettingsMapFromModel(settingsModel{})
	for _, k := range []string{"lifecycleEnabled", "lifecycleDefaultRunnerImage", "lifecycleMaxTimeoutSec"} {
		if v, ok := empty[k]; ok {
			t.Errorf("unset %s was included in the update map as %q", k, v)
		}
	}

	m := buildSettingsMapFromModel(settingsModel{
		LifecycleEnabled:            types.StringValue("true"),
		LifecycleDefaultRunnerImage: types.StringValue("ghcr.io/getsops/sops:v3.11.0"),
		LifecycleMaxTimeoutSec:      types.StringValue("300"),
	})
	want := map[string]string{
		"lifecycleEnabled":            "true",
		"lifecycleDefaultRunnerImage": "ghcr.io/getsops/sops:v3.11.0",
		"lifecycleMaxTimeoutSec":      "300",
	}
	if len(m) != len(want) {
		t.Errorf("update map carries unexpected extra keys: %v", m)
	}
	for k, w := range want {
		if got, ok := m[k]; !ok {
			t.Errorf("%s missing from update map", k)
		} else if got != w {
			t.Errorf("%s: got %q, want %q", k, got, w)
		}
	}
}

// TestAddWriteOnlySettings pins how the write-only secret settings reach the
// update map. They never appear in the plan, so they come from the
// configuration: all of them on create, and on update only those whose
// version changed (Terraform keeps no copy to diff the value itself).
func TestAddWriteOnlySettings(t *testing.T) {
	config := settingsModel{
		DepotTokenWO:              types.StringValue("depot"),
		DepotTokenWOVersion:       types.Int64Value(1),
		OidcClientSecretWO:        types.StringValue("oidc"),
		OidcClientSecretWOVersion: types.Int64Value(2),
		TrivyServerTokenWO:        types.StringValue("trivy"),
	}
	plan := config
	plan.DepotTokenWO = types.StringNull()
	plan.OidcClientSecretWO = types.StringNull()
	plan.TrivyServerTokenWO = types.StringNull()

	created := map[string]string{}
	addWriteOnlySettings(created, config, plan, nil)
	want := map[string]string{"depotToken": "depot", "oidcClientSecret": "oidc", "trivyServerToken": "trivy"}
	if len(created) != len(want) {
		t.Errorf("create map: got %v, want %v", created, want)
	}
	for k, w := range want {
		if created[k] != w {
			t.Errorf("create %s: got %q, want %q", k, created[k], w)
		}
	}

	// Only oidc_client_secret_wo_version moved (1 -> 2).
	prior := settingsModel{
		DepotTokenWOVersion:       types.Int64Value(1),
		OidcClientSecretWOVersion: types.Int64Value(1),
	}
	updated := map[string]string{}
	addWriteOnlySettings(updated, config, plan, &prior)
	if len(updated) != 1 || updated["oidcClientSecret"] != "oidc" {
		t.Errorf("update map: got %v, want only oidcClientSecret", updated)
	}
}
