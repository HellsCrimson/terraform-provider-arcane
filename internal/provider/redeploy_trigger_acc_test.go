package provider

import (
	"fmt"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// The redeploy_trigger acceptance tests all use last_redeploy as the observable
// side effect: the provider only stamps it when it actually called the redeploy
// endpoint, so "did this apply redeploy?" becomes an attribute assertion.

// TestAccArcaneProject_redeployTriggerNever verifies that redeploy_trigger =
// "never" suppresses the redeploy even when the compose content changes, and
// that the deprecated redeploy_on_update mirror follows the trigger.
func TestAccArcaneProject_redeployTriggerNever(t *testing.T) {
	name := testAccName("redeploy-never")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccProjectRedeployTriggerConfig(name, `redeploy_trigger = "never"`, "v1", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "redeploy_trigger", "never"),
					resource.TestCheckResourceAttr("arcane_project.test", "redeploy_on_update", "false"),
					resource.TestCheckNoResourceAttr("arcane_project.test", "last_redeploy"),
				),
			},
			{
				Config: testAccProjectRedeployTriggerConfig(name, `redeploy_trigger = "never"`, "v2", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "redeploy_trigger", "never"),
					resource.TestCheckNoResourceAttr("arcane_project.test", "last_redeploy"),
				),
			},
		},
	})
}

// TestAccArcaneProject_redeployTriggerDefault verifies that redeploy_trigger =
// "default" redeploys on a compose/env content change only, and leaves other
// in-place updates alone.
func TestAccArcaneProject_redeployTriggerDefault(t *testing.T) {
	name := testAccName("redeploy-default")
	var afterContentChange string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccProjectRedeployTriggerConfig(name, `redeploy_trigger = "default"`, "v1", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "redeploy_trigger", "default"),
					resource.TestCheckResourceAttr("arcane_project.test", "redeploy_on_update", "true"),
					resource.TestCheckNoResourceAttr("arcane_project.test", "last_redeploy"),
				),
			},
			{
				// Non-content change: no redeploy.
				Config: testAccProjectRedeployTriggerConfig(name, `redeploy_trigger = "default"`, "v1", "\n  remove_orphans = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "remove_orphans", "true"),
					resource.TestCheckNoResourceAttr("arcane_project.test", "last_redeploy"),
				),
			},
			{
				// Content change: redeploy.
				Config: testAccProjectRedeployTriggerConfig(name, `redeploy_trigger = "default"`, "v2", "\n  remove_orphans = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arcane_project.test", "last_redeploy"),
					testAccCaptureAttr("arcane_project.test", "last_redeploy", &afterContentChange),
				),
			},
			{
				// Non-content change again: the stamp must not move.
				Config: testAccProjectRedeployTriggerConfig(name, `redeploy_trigger = "default"`, "v2", "\n  remove_orphans = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAttrUnchanged("arcane_project.test", "last_redeploy", &afterContentChange),
				),
			},
		},
	})
}

// TestAccArcaneProject_redeployTriggerUpdate verifies that redeploy_trigger =
// "update" redeploys on any in-place update, including ones that leave the
// compose/env content untouched.
func TestAccArcaneProject_redeployTriggerUpdate(t *testing.T) {
	name := testAccName("redeploy-update")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccProjectRedeployTriggerConfig(name, `redeploy_trigger = "update"`, "v1", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "redeploy_trigger", "update"),
					resource.TestCheckNoResourceAttr("arcane_project.test", "last_redeploy"),
				),
			},
			{
				Config: testAccProjectRedeployTriggerConfig(name, `redeploy_trigger = "update"`, "v1", "\n  remove_orphans = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arcane_project.test", "last_redeploy"),
				),
			},
		},
	})
}

// TestAccArcaneProject_redeployTriggerAlways verifies that redeploy_trigger =
// "always" redeploys on every apply. Terraform only calls Update when the plan
// is non-empty, so the provider marks last_redeploy unknown during plan: the
// resulting perpetual diff is the mechanism, hence ExpectNonEmptyPlan.
func TestAccArcaneProject_redeployTriggerAlways(t *testing.T) {
	name := testAccName("redeploy-always")
	var firstRedeploy string

	cfg := testAccProjectRedeployTriggerConfig(name, `redeploy_trigger = "always"`, "v1", "")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				// Create deploys through "up", so nothing is redeployed yet.
				Config:             cfg,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "redeploy_trigger", "always"),
					resource.TestCheckNoResourceAttr("arcane_project.test", "last_redeploy"),
				),
			},
			{
				// Identical config: the apply still redeploys.
				Config:             cfg,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arcane_project.test", "last_redeploy"),
					testAccCaptureAttr("arcane_project.test", "last_redeploy", &firstRedeploy),
				),
			},
			{
				// And again, with a new timestamp each time.
				Config:             cfg,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAttrChanged("arcane_project.test", "last_redeploy", &firstRedeploy),
				),
			},
		},
	})
}

// TestAccArcaneProject_redeployOnUpdateLegacy verifies that the deprecated
// redeploy_on_update attribute keeps working: existing configurations must not
// break, and the value they set has to be reflected in the new attribute.
func TestAccArcaneProject_redeployOnUpdateLegacy(t *testing.T) {
	name := testAccName("redeploy-legacy")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccProjectRedeployTriggerConfig(name, `redeploy_on_update = false`, "v1", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "redeploy_on_update", "false"),
					resource.TestCheckResourceAttr("arcane_project.test", "redeploy_trigger", "never"),
				),
			},
			{
				// Content change with the legacy opt-out: still no redeploy.
				Config: testAccProjectRedeployTriggerConfig(name, `redeploy_on_update = false`, "v2", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("arcane_project.test", "last_redeploy"),
				),
			},
			{
				// Flipping the legacy attribute back on maps to "default".
				Config: testAccProjectRedeployTriggerConfig(name, `redeploy_on_update = true`, "v3", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "redeploy_trigger", "default"),
					resource.TestCheckResourceAttrSet("arcane_project.test", "last_redeploy"),
				),
			},
		},
	})
}

// TestAccArcaneProject_redeployTriggerInvalidConfigs covers the two ways of
// configuring the trigger wrong: an unsupported value, and setting both the new
// and the deprecated attribute (which would be ambiguous).
func TestAccArcaneProject_redeployTriggerInvalidConfigs(t *testing.T) {
	name := testAccName("redeploy-invalid")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config:      testAccProjectRedeployTriggerConfig(name, `redeploy_trigger = "sometimes"`, "v1", ""),
				ExpectError: regexp.MustCompile(`Invalid Attribute Value Match`),
			},
			{
				Config: testAccProjectRedeployTriggerConfig(name,
					"redeploy_trigger   = \"always\"\n  redeploy_on_update = true", "v1", ""),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}

func testAccProjectRedeployTriggerConfig(name, trigger, contentVersion, extra string) string {
	return fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}

resource "arcane_project" "test" {
  environment_id  = %q
  name            = %q
  compose_content = <<YAML
services:
  app:
    image: alpine:latest
    command: ["sh", "-c", "echo %s && sleep 300"]
YAML
  running         = true
  pull_on_update  = false
  remove_files    = true
  remove_volumes  = true
  %s%s
}
`, testAccEndpoint(), testAccAPIKey(), testAccEnvironmentID(), name, contentVersion, trigger, extra)
}

// TestAccArcaneProjectPath_redeployTrigger covers the same behaviour on
// arcane_project_path, where the content lives in files on disk: "never" must
// suppress the redeploy a file change would otherwise cause, and "always" must
// redeploy even when the files are untouched.
func TestAccArcaneProjectPath_redeployTrigger(t *testing.T) {
	name := testAccName("redeploy-project-path")
	var firstRedeploy string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				PreConfig: func() { writeProjectPathFixtures(t, "1") },
				Config:    testAccProjectPathRedeployTriggerConfig(name, "never"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project_path.test", "redeploy_trigger", "never"),
					resource.TestCheckNoResourceAttr("arcane_project_path.test", "last_redeploy"),
				),
			},
			{
				// File content changed, but the trigger says never.
				PreConfig: func() { writeProjectPathFixtures(t, "2") },
				Config:    testAccProjectPathRedeployTriggerConfig(name, "never"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("arcane_project_path.test", "last_redeploy"),
				),
			},
			{
				// Switching to always redeploys on this apply and every one after.
				PreConfig:          func() { writeProjectPathFixtures(t, "2") },
				Config:             testAccProjectPathRedeployTriggerConfig(name, "always"),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project_path.test", "redeploy_trigger", "always"),
					resource.TestCheckResourceAttrSet("arcane_project_path.test", "last_redeploy"),
					testAccCaptureAttr("arcane_project_path.test", "last_redeploy", &firstRedeploy),
				),
			},
			{
				PreConfig:          func() { writeProjectPathFixtures(t, "2") },
				Config:             testAccProjectPathRedeployTriggerConfig(name, "always"),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAttrChanged("arcane_project_path.test", "last_redeploy", &firstRedeploy),
				),
			},
		},
	})
}

func testAccProjectPathRedeployTriggerConfig(name, trigger string) string {
	return fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}

resource "arcane_project_path" "test" {
  environment_id    = %q
  name              = %q
  compose_path      = %q
  env_path          = %q
  content_hash_mode = true
  running           = true
  pull_on_update    = false
  redeploy_trigger  = %q
  remove_files      = true
  remove_volumes    = true
}
`, testAccEndpoint(), testAccAPIKey(), testAccEnvironmentID(), name,
		filepath.Join(testAccFixtureDirPath(), "docker-compose.yml"),
		filepath.Join(testAccFixtureDirPath(), ".env"),
		trigger)
}

// testAccCaptureAttr records an attribute value so a later step can compare
// against it. Absent (null) attributes are captured as "".
func testAccCaptureAttr(resourceName, attr string, dst *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		*dst = rs.Primary.Attributes[attr]
		return nil
	}
}

func testAccCheckAttrChanged(resourceName, attr string, previous *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		got := rs.Primary.Attributes[attr]
		if got == "" {
			return fmt.Errorf("%s.%s is not set", resourceName, attr)
		}
		if got == *previous {
			return fmt.Errorf("%s.%s did not change, still %q", resourceName, attr, got)
		}
		*previous = got
		return nil
	}
}

func testAccCheckAttrUnchanged(resourceName, attr string, previous *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		if got := rs.Primary.Attributes[attr]; got != *previous {
			return fmt.Errorf("%s.%s changed from %q to %q", resourceName, attr, *previous, got)
		}
		return nil
	}
}
