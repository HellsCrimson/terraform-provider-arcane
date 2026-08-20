package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Arcane refuses to rename a project that is not stopped:
//
//	arcane API error: 400 Bad Request: {... "detail":"Failed to update project:
//	project must be stopped before renaming (current status: running)"}
//
// The tests below pin down that the provider reports that requirement during
// the plan instead of letting the apply walk into the API error, and that the
// two ways of renaming a running project in a single apply work.

// TestAccArcaneProject_renameRunningFailsPlan is the regression test for issue
// #23: renaming a running project has to fail during the plan phase, not
// halfway through the apply. The step is PlanOnly on purpose — the apply is
// never reached, so an error can only come from the plan.
func TestAccArcaneProject_renameRunningFailsPlan(t *testing.T) {
	name := testAccName("rename-running")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccProjectRenameConfig(name, "  running = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "name", name),
					resource.TestCheckResourceAttr("arcane_project.test", "status", "running"),
				),
			},
			{
				Config:      testAccProjectRenameConfig(name+"-renamed", "  running = true"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`rename requires a stopped project`),
			},
		},
	})
}

// TestAccArcaneProject_renameStoppedInSameApply covers the rename flows that
// must keep working: stopping the project and renaming it in one apply (the
// provider has to bring it down before it calls the update endpoint), and
// renaming a project that is already stopped.
func TestAccArcaneProject_renameStoppedInSameApply(t *testing.T) {
	name := testAccName("rename-stop")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccProjectRenameConfig(name, "  running = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "name", name),
					resource.TestCheckResourceAttr("arcane_project.test", "status", "running"),
				),
			},
			{
				// Rename and stop in the same change: the plan must not fail,
				// and the apply must stop the project before renaming it.
				Config: testAccProjectRenameConfig(name+"-stopped", "  running = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "name", name+"-stopped"),
					resource.TestCheckResourceAttr("arcane_project.test", "running", "false"),
					resource.TestCheckResourceAttr("arcane_project.test", "status", "stopped"),
				),
			},
			{
				// Renaming an already stopped project needs no ceremony.
				Config: testAccProjectRenameConfig(name+"-again", "  running = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "name", name+"-again"),
					resource.TestCheckResourceAttr("arcane_project.test", "status", "stopped"),
				),
			},
		},
	})
}

// TestAccArcaneProject_renameStopBeforeRename covers the opt-in: with
// stop_before_rename = true the provider stops the project, renames it and
// starts it again within a single apply, so the project keeps running.
func TestAccArcaneProject_renameStopBeforeRename(t *testing.T) {
	name := testAccName("rename-optin")
	const optIn = "  running            = true\n  stop_before_rename = true"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccProjectRenameConfig(name, optIn),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "name", name),
					resource.TestCheckResourceAttr("arcane_project.test", "stop_before_rename", "true"),
					resource.TestCheckResourceAttr("arcane_project.test", "status", "running"),
				),
			},
			{
				Config: testAccProjectRenameConfig(name+"-renamed", optIn),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "name", name+"-renamed"),
					resource.TestCheckResourceAttr("arcane_project.test", "running", "true"),
					resource.TestCheckResourceAttr("arcane_project.test", "status", "running"),
				),
			},
		},
	})
}

// testAccProjectRenameConfig renders an arcane_project whose compose content
// never changes, so every step after the first plans a rename and nothing else.
// redeploy_trigger = "never" keeps redeploys out of the picture.
func testAccProjectRenameConfig(name, lifecycle string) string {
	return fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}

resource "arcane_project" "test" {
  environment_id   = %q
  name             = %q
  compose_content  = <<YAML
services:
  app:
    image: alpine:latest
    command: ["sh", "-c", "echo rename && sleep 300"]
YAML
  redeploy_trigger = "never"
  pull_on_update   = false
  remove_files     = true
  remove_volumes   = true
%s
}
`, testAccEndpoint(), testAccAPIKey(), testAccEnvironmentID(), name, lifecycle)
}

// TestAccArcaneProjectPath_renameRunningFailsPlan is the arcane_project_path
// half of issue #23: the rename requirement is a property of the Arcane API, so
// it has to be reported during the plan for this resource too.
func TestAccArcaneProjectPath_renameRunningFailsPlan(t *testing.T) {
	name := testAccName("rename-path")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				PreConfig: func() { writeProjectPathRenameFixtures(t) },
				Config:    testAccProjectPathRenameConfig(name, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project_path.test", "name", name),
					resource.TestCheckResourceAttr("arcane_project_path.test", "status", "running"),
				),
			},
			{
				PreConfig:   func() { writeProjectPathRenameFixtures(t) },
				Config:      testAccProjectPathRenameConfig(name+"-renamed", ""),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`rename requires a stopped project`),
			},
		},
	})
}

// TestAccArcaneProjectPath_renameStopBeforeRename covers the opt-in on
// arcane_project_path.
func TestAccArcaneProjectPath_renameStopBeforeRename(t *testing.T) {
	name := testAccName("rename-path-optin")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				PreConfig: func() { writeProjectPathRenameFixtures(t) },
				Config:    testAccProjectPathRenameConfig(name, "\n  stop_before_rename = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project_path.test", "name", name),
					resource.TestCheckResourceAttr("arcane_project_path.test", "status", "running"),
				),
			},
			{
				PreConfig: func() { writeProjectPathRenameFixtures(t) },
				Config:    testAccProjectPathRenameConfig(name+"-renamed", "\n  stop_before_rename = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project_path.test", "name", name+"-renamed"),
					resource.TestCheckResourceAttr("arcane_project_path.test", "status", "running"),
				),
			},
		},
	})
}

// writeProjectPathRenameFixtures writes compose/env fixtures whose service
// stays up long enough for the rename to run against a *running* project; the
// shared writeProjectPathFixtures service exits after a second.
func writeProjectPathRenameFixtures(t *testing.T) {
	t.Helper()

	dir := testAccFixtureDir(t)
	compose := `services:
  app:
    image: alpine:latest
    command: ["sh", "-c", "echo project-path-rename && sleep 300"]
`
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0o600); err != nil {
		t.Fatalf("writing compose fixture: %s", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("TFACC_SUFFIX=rename\n"), 0o600); err != nil {
		t.Fatalf("writing env fixture: %s", err)
	}
}

func testAccProjectPathRenameConfig(name, extra string) string {
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
  redeploy_trigger  = "never"
  remove_files      = true
  remove_volumes    = true%s
}
`, testAccEndpoint(), testAccAPIKey(), testAccEnvironmentID(), name,
		filepath.Join(testAccFixtureDirPath(), "docker-compose.yml"),
		filepath.Join(testAccFixtureDirPath(), ".env"),
		extra)
}
