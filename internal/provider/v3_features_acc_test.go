package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccProviderBlock() string {
	return fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}
`, testAccEndpoint(), testAccAPIKey())
}

// TestAccArcaneGlobalVariable covers a plain and a write-only secret variable:
// create, scope change, value rotation and import.
func TestAccArcaneGlobalVariable(t *testing.T) {
	testAccPreCheck(t)
	key := strings.ToUpper(strings.ReplaceAll(testAccName("var"), "-", "_"))
	config := func(value string, version int, scoped bool) string {
		scope := ""
		if scoped {
			scope = fmt.Sprintf("environment_ids = [%q]\n  inject_into_containers = true", testAccEnvironmentID())
		}
		return testAccProviderBlock() + fmt.Sprintf(`
resource "arcane_global_variable" "plain" {
  key   = "%[1]s_PLAIN"
  value = %[2]q
  %[4]s
}

resource "arcane_global_variable" "secret" {
  key              = "%[1]s_SECRET"
  value_wo         = "s3cr3t-%[3]d"
  value_wo_version = %[3]d
}
`, key, value, version, scope)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("one", 1, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_global_variable.plain", "value", "one"),
					resource.TestCheckResourceAttr("arcane_global_variable.plain", "all_environments", "true"),
					resource.TestCheckResourceAttr("arcane_global_variable.plain", "is_secret", "false"),
					resource.TestCheckResourceAttr("arcane_global_variable.secret", "is_secret", "true"),
					resource.TestCheckNoResourceAttr("arcane_global_variable.secret", "value_wo"),
				),
			},
			{
				Config: config("two", 2, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_global_variable.plain", "value", "two"),
					resource.TestCheckResourceAttr("arcane_global_variable.plain", "all_environments", "false"),
					resource.TestCheckResourceAttr("arcane_global_variable.plain", "environment_ids.#", "1"),
					resource.TestCheckResourceAttr("arcane_global_variable.plain", "inject_into_containers", "true"),
				),
			},
			{
				ResourceName:      "arcane_global_variable.plain",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccArcaneGitRepository_forgeAndAuthor sets, changes and clears the
// forge and commit author settings.
func TestAccArcaneGitRepository_forgeAndAuthor(t *testing.T) {
	testAccPreCheck(t)
	name := testAccName("forge-repo")
	config := func(extra string) string {
		return testAccProviderBlock() + fmt.Sprintf(`
resource "arcane_git_repository" "forge" {
  name      = %q
  url       = "https://github.com/docker/awesome-compose.git"
  auth_type = "none"
  %s
}
`, name, extra)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(`git_provider = "github"
  commit_author_name  = "Arcane Bot"
  commit_author_email = "bot@example.com"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_git_repository.forge", "git_provider", "github"),
					resource.TestCheckResourceAttr("arcane_git_repository.forge", "commit_author_name", "Arcane Bot"),
					resource.TestCheckResourceAttr("arcane_git_repository.forge", "has_signing_key", "false"),
				),
			},
			{
				Config: config(`commit_author_name = "Other Bot"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("arcane_git_repository.forge", "git_provider"),
					resource.TestCheckResourceAttr("arcane_git_repository.forge", "commit_author_name", "Other Bot"),
					resource.TestCheckNoResourceAttr("arcane_git_repository.forge", "commit_author_email"),
				),
			},
		},
	})
}

// TestAccArcaneGitOpsSync_backupMode backs up an existing project and checks
// that destroying the sync leaves the project in place.
func TestAccArcaneGitOpsSync_backupMode(t *testing.T) {
	testAccPreCheck(t)
	name := testAccName("backup-sync")
	client := sdkclient.NewClient(testAccEndpoint(), testAccAPIKey())
	var projectID string
	config := func(onSave bool, paths string) string {
		return testAccProviderBlock() + fmt.Sprintf(`
resource "arcane_git_repository" "backup" {
  name      = "%[1]s-repo"
  url       = "https://github.com/docker/awesome-compose.git"
  auth_type = "none"
}

resource "arcane_project" "backup" {
  environment_id   = %[2]q
  name             = "%[1]s-project"
  compose_content  = "services:\n  app:\n    image: alpine:latest\n"
  redeploy_trigger = "never"
  remove_files     = true
}

resource "arcane_gitops_sync" "backup" {
  environment_id   = %[2]q
  name             = %[1]q
  repository_id    = arcane_git_repository.backup.id
  branch           = "master"
  mode             = "backup"
  project_id       = arcane_project.backup.id
  backup_directory = "backups/%[1]s"
  backup_on_save   = %[3]t
  backup_paths     = %[4]s
  auto_sync        = false
}
`, name, testAccEnvironmentID(), onSave, paths)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(false, `["compose.yaml"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_gitops_sync.backup", "mode", "backup"),
					resource.TestCheckResourceAttr("arcane_gitops_sync.backup", "backup_on_save", "false"),
					resource.TestCheckResourceAttr("arcane_gitops_sync.backup", "compose_path", "backups/"+name+"/compose.yaml"),
					resource.TestCheckResourceAttrPair("arcane_gitops_sync.backup", "project_id", "arcane_project.backup", "id"),
					func(s *terraform.State) error {
						projectID = s.RootModule().Resources["arcane_project.backup"].Primary.ID
						return nil
					},
				),
			},
			{
				Config: config(true, `["compose.yaml", ".env"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_gitops_sync.backup", "backup_on_save", "true"),
					resource.TestCheckResourceAttr("arcane_gitops_sync.backup", "backup_paths.#", "2"),
				),
			},
			// Drop the sync alone: the project it backed up must survive.
			{
				Config: testAccProviderBlock() + fmt.Sprintf(`
resource "arcane_git_repository" "backup" {
  name      = "%[1]s-repo"
  url       = "https://github.com/docker/awesome-compose.git"
  auth_type = "none"
}

resource "arcane_project" "backup" {
  environment_id   = %[2]q
  name             = "%[1]s-project"
  compose_content  = "services:\n  app:\n    image: alpine:latest\n"
  redeploy_trigger = "never"
  remove_files     = true
}
`, name, testAccEnvironmentID()),
				Check: func(*terraform.State) error {
					if _, err := client.GetProject(context.Background(), testAccEnvironmentID(), projectID); err != nil {
						return fmt.Errorf("project backed up by the destroyed sync is gone: %w", err)
					}
					return nil
				},
			},
		},
	})
}

// TestAccArcaneV3_resourceOptions creates the network, container, project,
// volume backup and settings options added for Arcane v3.
func TestAccArcaneV3_resourceOptions(t *testing.T) {
	testAccPreCheck(t)
	name := testAccName("v3opts")
	env := testAccEnvironmentID()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderBlock() + fmt.Sprintf(`
resource "arcane_network" "ipam" {
  environment_id = %[2]q
  name           = "%[1]s-net"
  ipam = {
    config = [{
      subnet  = "172.31.240.0/24"
      gateway = "172.31.240.1"
    }]
  }
}

resource "arcane_volume" "data" {
  environment_id = %[2]q
  name           = "%[1]s-vol"
}

resource "arcane_container" "hc" {
  environment_id  = %[2]q
  name            = "%[1]s-ctr"
  image           = "alpine:latest"
  command         = ["sleep", "300"]
  cap_drop        = ["ALL"]
  readonly_rootfs = true
  restart_policy      = "on-failure"
  restart_max_retries = 2
  mounts = [{
    type   = "volume"
    source = arcane_volume.data.name
    target = "/data"
  }]
  healthcheck = {
    test     = ["CMD", "true"]
    interval = 30
    retries  = 3
  }
  force_delete = true
}

resource "arcane_project" "opts" {
  environment_id   = %[2]q
  name             = "%[1]s-project"
  compose_content  = "services:\n  app:\n    image: alpine:latest\n    command: [\"sleep\", \"300\"]\n"
  running          = true
  build            = false
  recreate_volumes = true
  redeploy_trigger = "never"
  remove_files     = true
}

resource "arcane_volume_backup" "local" {
  environment_id = %[2]q
  volume_name    = arcane_volume.data.name
  destination    = "local"
}

resource "arcane_settings" "v3" {
  environment_id               = %[2]q
  build_history_retention_days = "14"
}
`, name, env),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_network.ipam", "ipam.config.0.subnet", "172.31.240.0/24"),
					resource.TestCheckResourceAttrSet("arcane_container.hc", "id"),
					resource.TestCheckResourceAttr("arcane_project.opts", "recreate_volumes", "true"),
					resource.TestCheckResourceAttr("arcane_volume_backup.local", "destination", "local"),
					resource.TestCheckResourceAttrSet("arcane_volume_backup.local", "status"),
					resource.TestCheckResourceAttr("arcane_settings.v3", "applied.buildHistoryRetentionDays", "14"),
				),
			},
		},
	})
}
