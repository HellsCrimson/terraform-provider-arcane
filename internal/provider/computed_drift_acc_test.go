package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Server-derived computed attributes (status, running_count, ...) are written
// from the API response during Update, but nothing tells Terraform they may
// move: the framework turns computed attributes into "known after apply" only
// when the *configuration* changed something. A plan that exists solely because
// ModifyPlan produced it therefore carries the prior state's values, and the
// update fails with
//
//	Error: Provider produced inconsistent result after apply
//	... produced an unexpected new value: .status: was cty.StringVal("stopped"),
//	but now cty.StringVal("running").
//
// Those ModifyPlan-only plans are not a corner case: they are how
// arcane_project_path normally works (the compose file changes while the
// configuration does not) and how redeploy_trigger = "always" works on both
// resources.
//
// Both tests below deploy a container that exits on its own, let it die, and
// then drive an update that redeploys. The redeploy is what makes the failure
// deterministic: it brings the project back up, so status moves during the apply
// itself and no amount of refreshing beforehand could have predicted it.

// computedDriftSettleTime is how long the tests wait for the container deployed
// by the previous step to exit. Comfortably longer than the container's own
// lifetime (see computedDriftCompose).
const computedDriftSettleTime = 8 * time.Second

// computedDriftCompose is a project whose single container exits a few seconds
// after it is deployed.
func computedDriftCompose(version string) string {
	return fmt.Sprintf(`services:
  app:
    image: alpine:latest
    command: ["sh", "-c", "echo %s && sleep 3"]
`, version)
}

// TestAccArcaneProjectPath_computedDriftOnUpdate is the realistic case: the
// compose file on disk changed, which only ModifyPlan can see, so Terraform
// plans no change to status. The content change then redeploys the project,
// which brings the exited container back up.
func TestAccArcaneProjectPath_computedDriftOnUpdate(t *testing.T) {
	name := testAccName("computed-drift-path")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				PreConfig: func() { writeProjectPathDriftFixtures(t, "v1") },
				Config:    testAccProjectPathComputedDriftConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project_path.test", "status", "running"),
				),
			},
			{
				// Identical configuration: the only change is on disk. The
				// container has exited by the time the update runs, so the plan
				// is built against a stopped project that the redeploy restarts.
				PreConfig: func() {
					time.Sleep(computedDriftSettleTime)
					writeProjectPathDriftFixtures(t, "v2")
				},
				Config: testAccProjectPathComputedDriftConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project_path.test", "status", "running"),
					resource.TestCheckResourceAttrSet("arcane_project_path.test", "last_redeploy"),
				),
			},
		},
	})
}

func writeProjectPathDriftFixtures(t *testing.T, version string) {
	t.Helper()

	dir := testAccFixtureDir(t)
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(computedDriftCompose(version)), 0o600); err != nil {
		t.Fatalf("writing compose fixture: %s", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("TFACC_DRIFT="+version+"\n"), 0o600); err != nil {
		t.Fatalf("writing env fixture: %s", err)
	}
}

func testAccProjectPathComputedDriftConfig(name string) string {
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
  redeploy_trigger  = "default"
  remove_files      = true
  remove_volumes    = true
}
`, testAccEndpoint(), testAccAPIKey(), testAccEnvironmentID(), name,
		filepath.Join(testAccFixtureDirPath(), "docker-compose.yml"),
		filepath.Join(testAccFixtureDirPath(), ".env"),
	)
}

// TestAccArcaneProject_computedDriftOnUpdate is the same defect on
// arcane_project, reached through redeploy_trigger = "always": the plan exists
// only because ModifyPlan marked last_redeploy unknown, so every other computed
// attribute is planned as its prior value while the redeploy restarts the
// project underneath.
func TestAccArcaneProject_computedDriftOnUpdate(t *testing.T) {
	name := testAccName("computed-drift")
	cfg := testAccProjectComputedDriftConfig(name)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config:             cfg,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "status", "running"),
				),
			},
			{
				PreConfig:          func() { time.Sleep(computedDriftSettleTime) },
				Config:             cfg,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "status", "running"),
					resource.TestCheckResourceAttrSet("arcane_project.test", "last_redeploy"),
				),
			},
		},
	})
}

func testAccProjectComputedDriftConfig(name string) string {
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
%sYAML
  running          = true
  pull_on_update   = false
  redeploy_trigger = "always"
  remove_files     = true
  remove_volumes   = true
}
`, testAccEndpoint(), testAccAPIKey(), testAccEnvironmentID(), name, computedDriftCompose("drift"))
}
