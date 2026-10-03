package provider

import (
	"context"
	"fmt"
	"testing"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccArcaneGitOpsSync_afterSync covers redeploy_after_sync and
// pull_image_after_sync: both are sent on create, changed in place, and read
// back from the API.
func TestAccArcaneGitOpsSync_afterSync(t *testing.T) {
	name := testAccName("after-sync")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccGitOpsSyncAfterSyncConfig(name, true, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_gitops_sync.after_sync", "redeploy_after_sync", "true"),
					resource.TestCheckResourceAttr("arcane_gitops_sync.after_sync", "pull_image_after_sync", "false"),
					testAccCheckGitOpsSyncAfterSync("arcane_gitops_sync.after_sync", true, false),
				),
			},
			{
				Config: testAccGitOpsSyncAfterSyncConfig(name, false, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_gitops_sync.after_sync", "redeploy_after_sync", "false"),
					resource.TestCheckResourceAttr("arcane_gitops_sync.after_sync", "pull_image_after_sync", "true"),
					testAccCheckGitOpsSyncAfterSync("arcane_gitops_sync.after_sync", false, true),
				),
			},
		},
	})
}

// testAccCheckGitOpsSyncAfterSync reads the sync straight from the API, so the
// test fails if the provider only echoes the configuration into state.
func testAccCheckGitOpsSyncAfterSync(address string, redeploy, pull bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("%s not found in state", address)
		}
		client := sdkclient.NewClient(testAccEndpoint(), testAccAPIKey())
		sync, err := client.GetGitOpsSync(context.Background(), rs.Primary.Attributes["environment_id"], rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("reading gitops sync: %w", err)
		}
		if sync.RedeployAfterSync != redeploy {
			return fmt.Errorf("API redeployAfterSync = %t, want %t", sync.RedeployAfterSync, redeploy)
		}
		if sync.PullImageAfterSync != pull {
			return fmt.Errorf("API pullImageAfterSync = %t, want %t", sync.PullImageAfterSync, pull)
		}
		return nil
	}
}

func testAccGitOpsSyncAfterSyncConfig(name string, redeploy, pull bool) string {
	return fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}

resource "arcane_git_repository" "after_sync" {
  name      = %q
  url       = "https://github.com/docker/awesome-compose.git"
  auth_type = "none"
  enabled   = true
}

resource "arcane_gitops_sync" "after_sync" {
  environment_id = %q
  name           = %q
  repository_id  = arcane_git_repository.after_sync.id
  branch         = "master"
  compose_path   = "nginx-flask-mysql/compose.yaml"
  project_name   = %q
  auto_sync      = false
  target_type    = "project"
  start_project  = false

  redeploy_after_sync   = %t
  pull_image_after_sync = %t
}
`, testAccEndpoint(), testAccAPIKey(), testAccName("after-sync-repo"), testAccEnvironmentID(), name, name, redeploy, pull)
}
