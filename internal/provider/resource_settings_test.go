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
