package provider

import (
	"context"
	"fmt"
	"testing"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccArcaneSettings_detectsDrift is the end-to-end test for issue #37: a
// configured setting changed outside Terraform must show up in the next plan
// and be reverted on apply.
//
// Flow:
//  1. Apply auth_local_enabled = "false" and default_shell = "/bin/bash".
//  2. Change both through the API (PreConfig), as in the issue, and apply the
//     same configuration: the plan must be an update, and afterwards Arcane
//     must hold the configured values again.
//  3. Restore the instance defaults so local login stays enabled.
func TestAccArcaneSettings_detectsDrift(t *testing.T) {
	ctx := context.Background()
	client := sdkclient.NewClient(testAccEndpoint(), testAccAPIKey())
	envID := testAccEnvironmentID()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccSettingsDriftConfig("false", "/bin/bash"),
				Check:  testAccCheckSettingsAPI(client, envID, "false", "/bin/bash"),
			},
			{
				PreConfig: func() {
					if _, err := client.UpdateSettings(ctx, envID, map[string]string{
						"authLocalEnabled": "true",
						"defaultShell":     "/bin/zsh",
					}); err != nil {
						t.Fatalf("change settings out of band: %v", err)
					}
				},
				Config: testAccSettingsDriftConfig("false", "/bin/bash"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("arcane_settings.drift", plancheck.ResourceActionUpdate),
					},
				},
				Check: testAccCheckSettingsAPI(client, envID, "false", "/bin/bash"),
			},
			{
				Config: testAccSettingsDriftConfig("true", "/bin/sh"),
				Check:  testAccCheckSettingsAPI(client, envID, "true", "/bin/sh"),
			},
		},
	})
}

// testAccCheckSettingsAPI checks the values Arcane holds, not the state.
func testAccCheckSettingsAPI(client *sdkclient.Client, envID, authLocalEnabled, defaultShell string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		got, err := client.GetSettings(context.Background(), envID)
		if err != nil {
			return err
		}
		if got["authLocalEnabled"] != authLocalEnabled {
			return fmt.Errorf("authLocalEnabled: got %q, want %q", got["authLocalEnabled"], authLocalEnabled)
		}
		if got["defaultShell"] != defaultShell {
			return fmt.Errorf("defaultShell: got %q, want %q", got["defaultShell"], defaultShell)
		}
		return nil
	}
}

func testAccSettingsDriftConfig(authLocalEnabled, defaultShell string) string {
	return fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}

resource "arcane_settings" "drift" {
  environment_id     = %q
  auth_local_enabled = %q
  default_shell      = %q
}
`, testAccEndpoint(), testAccAPIKey(), testAccEnvironmentID(), authLocalEnabled, defaultShell)
}
