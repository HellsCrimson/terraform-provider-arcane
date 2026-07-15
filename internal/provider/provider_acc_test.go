package provider

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	testEndpoint = "http://localhost:3552/api"
	testAPIKey   = "arc_a54fe1040057252a19b34d72008395141a04de7731a28d6f7359baa4923b2f6a"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"arcane": providerserver.NewProtocol6WithError(New("test")()),
}

func TestAccArcaneProvider_allResources(t *testing.T) {
	// Start a webhook listener on the host that captures the POST Arcane sends when
	// the generic notification provider is tested. It is bound to all interfaces so
	// the Arcane container can reach it via host.docker.internal.
	webhook := startWebhookCapture(t)
	webhookURL1 := webhook.url + "?s=1"
	webhookURL2 := webhook.url + "?s=2"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					writeProjectPathFixtures(t, "1")
				},
				Config: testAccAllResourcesConfig("1", webhookURL1),
				Check: resource.ComposeAggregateTestCheckFunc(
					logCheck(t, "arcane_environment",
						resource.TestCheckResourceAttrSet("arcane_environment.test", "id"),
						resource.TestCheckResourceAttr("arcane_environment.test", "name", testAccName("env-1")),
					),
					logCheck(t, "arcane_user",
						resource.TestCheckResourceAttrSet("arcane_user.test", "id"),
						resource.TestCheckResourceAttr("arcane_user.test", "display_name", "Terraform Acceptance User 1"),
					),
					logCheck(t, "arcane_api_key", resource.TestCheckResourceAttrSet("arcane_api_key.test", "id")),
					logCheck(t, "arcane_container_registry", resource.TestCheckResourceAttrSet("arcane_container_registry.test", "id")),
					logCheck(t, "arcane_git_repository", resource.TestCheckResourceAttrSet("arcane_git_repository.test", "id")),
					logCheck(t, "arcane_template", resource.TestCheckResourceAttrSet("arcane_template.test", "id")),
					logCheck(t, "arcane_template_registry", resource.TestCheckResourceAttrSet("arcane_template_registry.test", "id")),
					logCheck(t, "arcane_project", resource.TestCheckResourceAttrSet("arcane_project.test", "id")),
					logCheck(t, "arcane_project_path", resource.TestCheckResourceAttrSet("arcane_project_path.test", "id")),
					logCheck(t, "arcane_container", resource.TestCheckResourceAttrSet("arcane_container.test", "id")),
					logCheck(t, "arcane_network", resource.TestCheckResourceAttrSet("arcane_network.test", "id")),
					logCheck(t, "arcane_volume", resource.TestCheckResourceAttrSet("arcane_volume.test", "id")),
					logCheck(t, "arcane_volume_backup", resource.TestCheckResourceAttrSet("arcane_volume_backup.test", "id")),
					logCheck(t, "arcane_vulnerability_ignore", resource.TestCheckResourceAttrSet("arcane_vulnerability_ignore.test", "id")),
					logCheck(t, "arcane_job_schedules", resource.TestCheckResourceAttr("arcane_job_schedules.test", "polling_interval", "0 */5 * * * *")),
					logCheck(t, "arcane_settings", resource.TestCheckResourceAttr("arcane_settings.test", "application_theme", "light")),
					logCheck(t, "arcane_notification",
						// Notification: generic webhook provider, enabled, pointing at the listener.
						resource.TestCheckResourceAttrSet("arcane_notification.test", "id"),
						resource.TestCheckResourceAttr("arcane_notification.test", "provider_name", "generic"),
						resource.TestCheckResourceAttr("arcane_notification.test", "enabled", "true"),
						resource.TestCheckResourceAttr("arcane_notification.test", "config.webhookUrl", webhookURL1),
						// Trigger a test notification and assert the listener received the POST.
						testAccCheckNotificationDelivers(webhook, testAccEnvironmentID()),
					),
					logCheck(t, "arcane_gitops_sync",
						resource.TestCheckResourceAttrSet("arcane_gitops_sync.test", "id"),
						resource.TestCheckResourceAttr("arcane_gitops_sync.test", "name", testAccName("gitops-1")),
						resource.TestCheckResourceAttr("arcane_gitops_sync.test", "branch", "master"),
						resource.TestCheckResourceAttr("arcane_gitops_sync.test", "target_type", "project"),
						resource.TestCheckResourceAttr("arcane_gitops_sync.test", "environment_variables.TFACC_SUFFIX", "1"),
					),
				),
			},
			{
				PreConfig: func() {
					writeProjectPathFixtures(t, "2")
				},
				Config: testAccAllResourcesConfig("2", webhookURL2),
				Check: resource.ComposeAggregateTestCheckFunc(
					logCheck(t, "arcane_environment", resource.TestCheckResourceAttr("arcane_environment.test", "name", testAccName("env-2"))),
					logCheck(t, "arcane_user", resource.TestCheckResourceAttr("arcane_user.test", "display_name", "Terraform Acceptance User 2")),
					logCheck(t, "arcane_api_key", resource.TestCheckResourceAttr("arcane_api_key.test", "name", testAccName("api-key-2"))),
					logCheck(t, "arcane_container_registry", resource.TestCheckResourceAttr("arcane_container_registry.test", "description", "Terraform acceptance registry 2")),
					logCheck(t, "arcane_git_repository", resource.TestCheckResourceAttr("arcane_git_repository.test", "description", "Terraform acceptance repository 2")),
					logCheck(t, "arcane_template", resource.TestCheckResourceAttr("arcane_template.test", "description", "Terraform acceptance template 2")),
					logCheck(t, "arcane_template_registry", resource.TestCheckResourceAttr("arcane_template_registry.test", "description", "Terraform acceptance template registry 2")),
					logCheck(t, "arcane_project", resource.TestCheckResourceAttr("arcane_project.test", "name", testAccName("project-2"))),
					logCheck(t, "arcane_project_path", resource.TestCheckResourceAttr("arcane_project_path.test", "name", testAccName("project-path-2"))),
					logCheck(t, "arcane_container", resource.TestCheckResourceAttr("arcane_container.test", "name", testAccName("container-2"))),
					logCheck(t, "arcane_network", resource.TestCheckResourceAttr("arcane_network.test", "name", testAccName("network-2"))),
					logCheck(t, "arcane_volume", resource.TestCheckResourceAttr("arcane_volume.test", "name", testAccName("volume-2"))),
					logCheck(t, "arcane_job_schedules", resource.TestCheckResourceAttr("arcane_job_schedules.test", "polling_interval", "0 */10 * * * *")),
					logCheck(t, "arcane_settings", resource.TestCheckResourceAttr("arcane_settings.test", "application_theme", "dark")),
					logCheck(t, "arcane_vulnerability_ignore", resource.TestCheckResourceAttr("arcane_vulnerability_ignore.test", "reason", "Terraform acceptance 2")),
					logCheck(t, "arcane_notification",
						// Notification update: disabled and webhook URL changed.
						resource.TestCheckResourceAttr("arcane_notification.test", "provider_name", "generic"),
						resource.TestCheckResourceAttr("arcane_notification.test", "enabled", "false"),
						resource.TestCheckResourceAttr("arcane_notification.test", "config.webhookUrl", webhookURL2),
					),
					logCheck(t, "arcane_gitops_sync",
						// GitOps sync update.
						resource.TestCheckResourceAttr("arcane_gitops_sync.test", "name", testAccName("gitops-2")),
						resource.TestCheckResourceAttr("arcane_gitops_sync.test", "environment_variables.TFACC_SUFFIX", "2"),
					),
				),
			},
			{
				// Data sources: read back the resources created above (suffix "2")
				// and assert each one mirrors its resource counterpart. The framework
				// destroys everything after this final step.
				PreConfig: func() {
					writeProjectPathFixtures(t, "2")
				},
				Config: testAccAllResourcesConfig("2", webhookURL2) + testAccDataSourcesConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					logCheck(t, "data.arcane_api_key",
						resource.TestCheckResourceAttrPair("data.arcane_api_key.test", "name", "arcane_api_key.test", "name"),
					),
					logCheck(t, "data.arcane_container",
						resource.TestCheckResourceAttrPair("data.arcane_container.test", "name", "arcane_container.test", "name"),
						resource.TestCheckResourceAttrPair("data.arcane_container.test", "image", "arcane_container.test", "image"),
					),
					logCheck(t, "data.arcane_container_registry",
						resource.TestCheckResourceAttrPair("data.arcane_container_registry.test", "url", "arcane_container_registry.test", "url"),
						resource.TestCheckResourceAttrPair("data.arcane_container_registry.test", "description", "arcane_container_registry.test", "description"),
					),
					logCheck(t, "data.arcane_environment",
						resource.TestCheckResourceAttrPair("data.arcane_environment.test", "name", "arcane_environment.test", "name"),
					),
					logCheck(t, "data.arcane_gitops_sync",
						resource.TestCheckResourceAttrPair("data.arcane_gitops_sync.test", "name", "arcane_gitops_sync.test", "name"),
						resource.TestCheckResourceAttrPair("data.arcane_gitops_sync.test", "branch", "arcane_gitops_sync.test", "branch"),
					),
					logCheck(t, "data.arcane_git_repository",
						resource.TestCheckResourceAttrPair("data.arcane_git_repository.test", "name", "arcane_git_repository.test", "name"),
						resource.TestCheckResourceAttrPair("data.arcane_git_repository.test", "url", "arcane_git_repository.test", "url"),
					),
					logCheck(t, "data.arcane_job_schedules",
						resource.TestCheckResourceAttrPair("data.arcane_job_schedules.test", "polling_interval", "arcane_job_schedules.test", "polling_interval"),
					),
					logCheck(t, "data.arcane_network",
						resource.TestCheckResourceAttrPair("data.arcane_network.test", "name", "arcane_network.test", "name"),
						resource.TestCheckResourceAttrPair("data.arcane_network.test", "driver", "arcane_network.test", "driver"),
					),
					logCheck(t, "data.arcane_notification",
						resource.TestCheckResourceAttrPair("data.arcane_notification.test", "provider_name", "arcane_notification.test", "provider_name"),
						resource.TestCheckResourceAttrPair("data.arcane_notification.test", "enabled", "arcane_notification.test", "enabled"),
					),
					logCheck(t, "data.arcane_project",
						resource.TestCheckResourceAttrPair("data.arcane_project.test", "name", "arcane_project.test", "name"),
					),
					logCheck(t, "data.arcane_project_path",
						resource.TestCheckResourceAttrPair("data.arcane_project_path.test", "name", "arcane_project_path.test", "name"),
					),
					logCheck(t, "data.arcane_settings",
						resource.TestCheckResourceAttr("data.arcane_settings.test", "settings.applicationTheme", "dark"),
					),
					logCheck(t, "data.arcane_template",
						resource.TestCheckResourceAttrPair("data.arcane_template.test", "name", "arcane_template.test", "name"),
					),
					logCheck(t, "data.arcane_template_registry",
						resource.TestCheckResourceAttrPair("data.arcane_template_registry.test", "name", "arcane_template_registry.test", "name"),
					),
					logCheck(t, "data.arcane_user",
						resource.TestCheckResourceAttrPair("data.arcane_user.test", "username", "arcane_user.test", "username"),
						resource.TestCheckResourceAttrPair("data.arcane_user.test", "display_name", "arcane_user.test", "display_name"),
					),
					logCheck(t, "data.arcane_volume",
						resource.TestCheckResourceAttrPair("data.arcane_volume.test", "name", "arcane_volume.test", "name"),
					),
				),
			},
		},
	})
}

// logCheck groups one or more checks under a label and, when they all pass,
// emits a single pass line to the test log (visible with `go test -v`). This
// makes the otherwise-monolithic acceptance test report a distinct line per
// resource / data source instead of one opaque pass/fail.
func logCheck(t *testing.T, label string, checks ...resource.TestCheckFunc) resource.TestCheckFunc {
	t.Helper()
	return func(s *terraform.State) error {
		for _, c := range checks {
			if err := c(s); err != nil {
				return fmt.Errorf("%s: %w", label, err)
			}
		}
		t.Logf("PASS %s", label)
		return nil
	}
}

// TestAccArcaneProject_failIfNameExists verifies the opt-in fail_if_name_exists
// guard on arcane_project. Arcane auto-renames a project (appending a numeric
// suffix) when one of the same name already exists, which is non-deterministic
// for Terraform. With fail_if_name_exists = true the provider instead fails the
// plan when a project of the same name is already present in the environment.
//
// The first project is created on its own (no collision, so the plan succeeds).
// A second project then reuses the same name; because the first project now
// exists, planning the second fails before any auto-rename can occur.
func TestAccArcaneProject_failIfNameExists(t *testing.T) {
	name := testAccName("dup-project")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccProjectFailIfNameExistsConfig(name, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arcane_project.first", "id"),
					resource.TestCheckResourceAttr("arcane_project.first", "name", name),
					resource.TestCheckResourceAttr("arcane_project.first", "fail_if_name_exists", "true"),
				),
			},
			{
				Config:      testAccProjectFailIfNameExistsConfig(name, true),
				ExpectError: regexp.MustCompile(`project name already exists`),
			},
		},
	})
}

func testAccProjectFailIfNameExistsConfig(name string, withSecond bool) string {
	cfg := fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}

resource "arcane_project" "first" {
  environment_id      = %q
  name                = %q
  compose_content     = <<YAML
services:
  app:
    image: alpine:latest
    command: ["sh", "-c", "echo dup && sleep 1"]
YAML
  running             = false
  redeploy_on_update  = false
  pull_on_update      = false
  fail_if_name_exists = true
  remove_files        = true
  remove_volumes      = true
}
`, testAccEndpoint(), testAccAPIKey(), testAccEnvironmentID(), name)

	if withSecond {
		cfg += fmt.Sprintf(`
resource "arcane_project" "second" {
  environment_id      = %q
  name                = %q
  compose_content     = <<YAML
services:
  app:
    image: alpine:latest
    command: ["sh", "-c", "echo dup2 && sleep 1"]
YAML
  running             = false
  redeploy_on_update  = false
  pull_on_update      = false
  fail_if_name_exists = true
  remove_files        = true
  remove_volumes      = true
}
`, testAccEnvironmentID(), name)
	}

	return cfg
}

// TestAccArcaneProject_addRemoveFilesToExisting is a regression test for issue
// #16: adding remove_files / remove_volumes to an already-created arcane_project
// failed with "Provider produced inconsistent result after apply:
// .remove_volumes: was cty.True, but now null".
//
// The project is first created WITHOUT remove_files / remove_volumes, so both
// land in state as null. A second step then adds remove_files = true and
// remove_volumes = true, which is an in-place Update. The Update handler must
// copy these planned values into the new state; if it does not, Terraform sees
// the planned value (true) replaced by null after apply and fails.
//
// Before the fix this second step errors; after the fix it applies cleanly and
// the attributes read back as true.
func TestAccArcaneProject_addRemoveFilesToExisting(t *testing.T) {
	name := testAccName("remove-files-project")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				// Create without the delete options: both stay null in state.
				Config: testAccProjectRemoveFilesConfig(name, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arcane_project.test", "id"),
					resource.TestCheckNoResourceAttr("arcane_project.test", "remove_files"),
					resource.TestCheckNoResourceAttr("arcane_project.test", "remove_volumes"),
				),
			},
			{
				// Add the delete options: in-place Update must persist them.
				Config: testAccProjectRemoveFilesConfig(name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "remove_files", "true"),
					resource.TestCheckResourceAttr("arcane_project.test", "remove_volumes", "true"),
				),
			},
		},
	})
}

func testAccProjectRemoveFilesConfig(name string, withRemoveOpts bool) string {
	removeOpts := ""
	if withRemoveOpts {
		removeOpts = `
  remove_files       = true
  remove_volumes     = true`
	}

	return fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}

resource "arcane_project" "test" {
  environment_id     = %q
  name               = %q
  compose_content    = <<YAML
services:
  app:
    image: alpine:latest
    command: ["sh", "-c", "echo remove-files && sleep 1"]
YAML
  running            = false
  redeploy_on_update = false
  pull_on_update     = false%s
}
`, testAccEndpoint(), testAccAPIKey(), testAccEnvironmentID(), name, removeOpts)
}

// TestAccArcaneContainer_failIfNameExists verifies the opt-in fail_if_name_exists
// guard on arcane_container. With fail_if_name_exists = true the provider fails
// the plan when a container of the same name already exists in the environment,
// instead of letting Arcane reject the create at apply time.
//
// The first container is created on its own (no collision). A second container
// then reuses the same name; because the first now exists, planning the second
// fails before any create is attempted.
func TestAccArcaneContainer_failIfNameExists(t *testing.T) {
	name := testAccName("dup-container")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccContainerFailIfNameExistsConfig(name, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arcane_container.first", "id"),
					resource.TestCheckResourceAttr("arcane_container.first", "name", name),
					resource.TestCheckResourceAttr("arcane_container.first", "fail_if_name_exists", "true"),
				),
			},
			{
				Config:      testAccContainerFailIfNameExistsConfig(name, true),
				ExpectError: regexp.MustCompile(`container name already exists`),
			},
		},
	})
}

func testAccContainerFailIfNameExistsConfig(name string, withSecond bool) string {
	cfg := fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}

resource "arcane_container" "first" {
  environment_id      = %q
  name                = %q
  image               = "alpine:latest"
  command             = ["sh", "-c", "sleep 60"]
  force_delete        = true
  remove_volumes      = false
  fail_if_name_exists = true
}
`, testAccEndpoint(), testAccAPIKey(), testAccEnvironmentID(), name)

	if withSecond {
		cfg += fmt.Sprintf(`
resource "arcane_container" "second" {
  environment_id      = %q
  name                = %q
  image               = "alpine:latest"
  command             = ["sh", "-c", "sleep 60"]
  force_delete        = true
  remove_volumes      = false
  fail_if_name_exists = true
}
`, testAccEnvironmentID(), name)
	}

	return cfg
}

// TestAccArcaneGitOpsSync_failIfNameExists verifies the opt-in fail_if_name_exists
// guard on arcane_gitops_sync. With fail_if_name_exists = true the provider fails
// the plan when a GitOps sync of the same name already exists in the environment,
// instead of creating a duplicate.
//
// The first sync is created on its own (no collision). A second sync then reuses
// the same name; because the first now exists, planning the second fails before
// any create is attempted.
func TestAccArcaneGitOpsSync_failIfNameExists(t *testing.T) {
	name := testAccName("dup-gitops")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccGitOpsSyncFailIfNameExistsConfig(name, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arcane_gitops_sync.first", "id"),
					resource.TestCheckResourceAttr("arcane_gitops_sync.first", "name", name),
					resource.TestCheckResourceAttr("arcane_gitops_sync.first", "fail_if_name_exists", "true"),
				),
			},
			{
				Config:      testAccGitOpsSyncFailIfNameExistsConfig(name, true),
				ExpectError: regexp.MustCompile(`gitops sync name already exists`),
			},
		},
	})
}

func testAccGitOpsSyncFailIfNameExistsConfig(name string, withSecond bool) string {
	cfg := fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}

resource "arcane_git_repository" "test" {
  name      = %q
  url       = "https://github.com/docker/awesome-compose.git"
  auth_type = "none"
  enabled   = true
}

resource "arcane_gitops_sync" "first" {
  environment_id      = %q
  name                = %q
  repository_id       = arcane_git_repository.test.id
  branch              = "master"
  compose_path        = "nginx-flask-mysql/compose.yaml"
  project_name        = "%s-1"
  auto_sync           = false
  sync_directory      = false
  target_type         = "project"
  start_project       = false
  fail_if_name_exists = true
}
`, testAccEndpoint(), testAccAPIKey(), testAccName("gitops-repo"), testAccEnvironmentID(), name, name)

	if withSecond {
		cfg += fmt.Sprintf(`
resource "arcane_gitops_sync" "second" {
  environment_id      = %q
  name                = %q
  repository_id       = arcane_git_repository.test.id
  branch              = "master"
  compose_path        = "nginx-flask-mysql/compose.yaml"
  project_name        = "%s-2"
  auto_sync           = false
  sync_directory      = false
  target_type         = "project"
  start_project       = false
  fail_if_name_exists = true
}
`, testAccEnvironmentID(), name, name)
	}

	return cfg
}

// TestAccArcaneEnvironmentID_forcesReplace verifies that changing environment_id
// on a per-environment resource is planned as a replacement (destroy + create)
// rather than an in-place update. environment_id is part of each resource's
// identity and the API has no "move between environments" operation, so an
// in-place update would leave the resource pointing at the old environment and
// produce a "provider produced inconsistent result after apply" error.
//
// Each resource is created in the live test environment, then a PlanOnly step
// flips environment_id to a different value and asserts the planned action is a
// replace. The bogus environment id is never contacted: the refresh reads the
// existing state (the real environment), and the plan is not applied.
func TestAccArcaneEnvironmentID_forcesReplace(t *testing.T) {
	cases := []struct {
		name     string
		addr     string
		configFn func(envID string) string
	}{
		{"project", "arcane_project.test", testAccEnvReplaceProjectConfig},
		{"container", "arcane_container.test", testAccEnvReplaceContainerConfig},
		{"notification", "arcane_notification.test", testAccEnvReplaceNotificationConfig},
		{"settings", "arcane_settings.test", testAccEnvReplaceSettingsConfig},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				PreCheck:                 func() { testAccPreCheck(t) },
				Steps: []resource.TestStep{
					{
						Config: tc.configFn(testAccEnvironmentID()),
						Check:  resource.TestCheckResourceAttr(tc.addr, "environment_id", testAccEnvironmentID()),
					},
					{
						Config:             tc.configFn("tfacc-nonexistent-environment"),
						PlanOnly:           true,
						ExpectNonEmptyPlan: true,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PostApplyPreRefresh: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(tc.addr, plancheck.ResourceActionReplace),
							},
						},
					},
				},
			})
		})
	}
}

func testAccEnvReplaceProviderBlock() string {
	return fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}
`, testAccEndpoint(), testAccAPIKey())
}

func testAccEnvReplaceProjectConfig(envID string) string {
	return testAccEnvReplaceProviderBlock() + fmt.Sprintf(`
resource "arcane_project" "test" {
  environment_id     = %q
  name               = %q
  compose_content    = <<YAML
services:
  app:
    image: alpine:latest
    command: ["sh", "-c", "sleep 1"]
YAML
  running            = false
  redeploy_on_update = false
  pull_on_update     = false
  remove_files       = true
  remove_volumes     = true
}
`, envID, testAccName("env-replace-project"))
}

func testAccEnvReplaceContainerConfig(envID string) string {
	return testAccEnvReplaceProviderBlock() + fmt.Sprintf(`
resource "arcane_container" "test" {
  environment_id = %q
  name           = %q
  image          = "alpine:latest"
  command        = ["sh", "-c", "sleep 60"]
  force_delete   = true
}
`, envID, testAccName("env-replace-container"))
}

func testAccEnvReplaceNotificationConfig(envID string) string {
	return testAccEnvReplaceProviderBlock() + fmt.Sprintf(`
resource "arcane_notification" "test" {
  environment_id = %q
  provider_name  = "generic"
  enabled        = false
  config = {
    webhookUrl = "http://example.com/hook"
  }
}
`, envID)
}

func testAccEnvReplaceSettingsConfig(envID string) string {
	return testAccEnvReplaceProviderBlock() + fmt.Sprintf(`
resource "arcane_settings" "test" {
  environment_id    = %q
  application_theme = "dark"
}
`, envID)
}

// webhookCapture is a host-side HTTP listener that records the POST requests
// Arcane sends when the generic notification provider is exercised.
type webhookCapture struct {
	mu       sync.Mutex
	requests [][]byte
	server   *httptest.Server
	url      string
}

func (w *webhookCapture) reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.requests = nil
}

func (w *webhookCapture) count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.requests)
}

// startWebhookCapture binds an HTTP server to all interfaces (so the Arcane
// container can reach it) and returns a handle whose url uses a host that is
// resolvable from inside the container.
func startWebhookCapture(t *testing.T) *webhookCapture {
	t.Helper()

	wc := &webhookCapture{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		wc.mu.Lock()
		wc.requests = append(wc.requests, body)
		wc.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("starting webhook listener: %s", err)
	}
	srv := httptest.NewUnstartedServer(handler)
	_ = srv.Listener.Close()
	srv.Listener = listener
	srv.Start()
	t.Cleanup(srv.Close)

	port := listener.Addr().(*net.TCPAddr).Port
	wc.server = srv
	wc.url = fmt.Sprintf("http://%s:%d/hook", testAccWebhookHost(), port)
	return wc
}

// testAccWebhookHost returns the host the Arcane container uses to reach the
// webhook listener running on the test host. Defaults to host.docker.internal,
// which the test docker-compose maps to the host gateway.
func testAccWebhookHost() string {
	if v := os.Getenv("ARCANE_ACC_WEBHOOK_HOST"); v != "" {
		return v
	}
	return "host.docker.internal"
}

// testAccCheckNotificationDelivers triggers a test notification through the
// generic provider and asserts the webhook listener received the resulting POST.
func testAccCheckNotificationDelivers(wc *webhookCapture, envID string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		wc.reset()
		client := sdkclient.NewClient(testAccEndpoint(), testAccAPIKey())
		if err := client.TestNotification(context.Background(), envID, "generic", "simple"); err != nil {
			return fmt.Errorf("triggering test notification: %w", err)
		}
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if wc.count() > 0 {
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
		return fmt.Errorf("webhook listener received no POST from Arcane; ensure the Arcane container can reach %q (override with ARCANE_ACC_WEBHOOK_HOST)", testAccWebhookHost())
	}
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	os.Setenv("ARCANE_ENDPOINT", testAccEndpoint())
	os.Setenv("ARCANE_API_KEY", testAccAPIKey())

	testAccRunID()
	testAccFixtureDir(t)
	testAccRestoreEnvironmentConfigOnCleanup(t)
}

func testAccEndpoint() string {
	if v := os.Getenv("ARCANE_ENDPOINT"); v != "" {
		return v
	}
	return testEndpoint
}

func testAccAPIKey() string {
	if v := os.Getenv("ARCANE_API_KEY"); v != "" {
		return v
	}
	return testAPIKey
}

func testAccEnvironmentID() string {
	if v := os.Getenv("ARCANE_ACC_ENVIRONMENT_ID"); v != "" {
		return v
	}
	return "0"
}

func testAccRestoreEnvironmentConfigOnCleanup(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	client := sdkclient.NewClient(testAccEndpoint(), testAccAPIKey())
	envID := testAccEnvironmentID()

	settings, err := client.GetSettings(ctx, envID)
	if err != nil {
		t.Fatalf("reading existing settings for cleanup snapshot: %s", err)
	}
	schedules, err := client.GetJobSchedules(ctx, envID)
	if err != nil {
		t.Fatalf("reading existing job schedules for cleanup snapshot: %s", err)
	}
	settingsToRestore := map[string]string{}
	for _, key := range []string{"applicationTheme", "pollingEnabled", "pollingInterval", "baseServerUrl"} {
		if value, ok := settings[key]; ok {
			settingsToRestore[key] = value
		}
	}

	t.Cleanup(func() {
		if len(settingsToRestore) > 0 {
			if _, err := client.UpdateSettings(context.Background(), envID, settingsToRestore); err != nil {
				t.Logf("failed to restore settings for environment %s: %s", envID, err)
			}
		}
		if _, err := client.UpdateJobSchedules(context.Background(), envID, sdkclient.UpdateJobSchedulesRequest{
			EnvironmentHealthInterval: stringPtr(schedules.EnvironmentHealthInterval),
			PollingInterval:           stringPtr(schedules.PollingInterval),
		}); err != nil {
			t.Logf("failed to restore job schedules for environment %s: %s", envID, err)
		}
	})
}

func stringPtr(v string) *string {
	return &v
}

func testAccRunID() string {
	if v := os.Getenv("ARCANE_ACC_RUN_ID"); v != "" {
		return v
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	v := fmt.Sprintf("%d%04d", time.Now().Unix(), r.Intn(10000))
	os.Setenv("ARCANE_ACC_RUN_ID", v)
	return v
}

func testAccName(prefix string) string {
	return fmt.Sprintf("tfacc-%s-%s", testAccRunID(), prefix)
}

func testAccFixtureDir(t *testing.T) string {
	t.Helper()

	dir := testAccFixtureDirPath()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("creating fixture directory: %s", err)
	}
	return dir
}

func testAccFixtureDirPath() string {
	if v := os.Getenv("ARCANE_ACC_FIXTURE_DIR"); v != "" {
		return v
	}
	return filepath.Join(os.TempDir(), "terraform-provider-arcane-acc-"+testAccRunID())
}

func writeProjectPathFixtures(t *testing.T, suffix string) {
	t.Helper()

	dir := testAccFixtureDir(t)
	compose := fmt.Sprintf(`services:
  app:
    image: alpine:latest
    command: ["sh", "-c", "echo project-path-%s && sleep 1"]
`, suffix)
	env := fmt.Sprintf("TFACC_SUFFIX=%s\n", suffix)

	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0o600); err != nil {
		t.Fatalf("writing compose fixture: %s", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0o600); err != nil {
		t.Fatalf("writing env fixture: %s", err)
	}
}

func testAccAllResourcesConfig(suffix, webhookURL string) string {
	envName := testAccName("env-" + suffix)
	return fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}

resource "arcane_environment" "test" {
  name        = %q
  api_url     = "http://localhost:3552"
  enabled     = true
  use_api_key = false
}

resource "arcane_user" "test" {
  username     = %q
  password     = "Terraform1!"
  display_name = "Terraform Acceptance User %s"
  email        = "tfacc-%s-%s@example.test"
  locale       = "en-US"
}

resource "arcane_api_key" "test" {
  name        = %q
  description = "Terraform acceptance API key %s"
  permissions = [
    {
      permission = "containers:list"
    },
  ]
}

resource "arcane_container_registry" "test" {
  url           = "https://example.test"
  username      = "tfacc"
  token         = "tfacc-token-%s"
  description   = "Terraform acceptance registry %s"
  insecure      = true
  enabled       = true
  registry_type = "generic"
}

resource "arcane_git_repository" "test" {
  name        = %q
  url         = "https://github.com/docker/awesome-compose.git"
  auth_type   = "none"
  description = "Terraform acceptance repository %s"
  enabled     = true
}

resource "arcane_template" "test" {
  name        = %q
  description = "Terraform acceptance template %s"
  content     = <<YAML
services:
  app:
    image: alpine:latest
    command: ["sh", "-c", "echo template-%s"]
YAML
  env_content = "TFACC_SUFFIX=%s\n"
}

resource "arcane_template_registry" "test" {
  name        = %q
  url         = "https://github.com/docker/awesome-compose"
  description = "Terraform acceptance template registry %s"
  enabled     = true
}

resource "arcane_settings" "test" {
  environment_id     = %q
  application_theme  = %q
  polling_enabled    = "false"
  polling_interval   = %q
  base_server_url    = "http://localhost:3552"
}

resource "arcane_job_schedules" "test" {
  environment_id              = %q
  polling_interval            = %q
  environment_health_interval = "0 */15 * * * *"
}

resource "arcane_notification" "test" {
  environment_id = %q
  provider_name  = "generic"
  enabled        = %s
  config = {
    webhookUrl = %q
  }
}

resource "arcane_project" "test" {
  environment_id      = %q
  name                = %q
  compose_content     = <<YAML
services:
  app:
    image: alpine:latest
    command: ["sh", "-c", "echo project-%s && sleep 1"]
YAML
  env_content         = "TFACC_SUFFIX=%s\n"
  running             = false
  redeploy_on_update  = false
  pull_on_update      = false
  remove_files        = true
  remove_volumes      = true
}

resource "arcane_project_path" "test" {
  environment_id     = %q
  name               = %q
  compose_path       = %q
  env_path           = %q
  content_hash_mode  = true
  running            = false
  pull_on_update     = false
  remove_files       = true
  remove_volumes     = true
}

resource "arcane_network" "test" {
  environment_id   = %q
  name             = %q
  driver           = "bridge"
  attachable       = true
  check_duplicate  = true
  labels = {
    tfacc = %q
  }
}

resource "arcane_volume" "test" {
  environment_id = %q
  name           = %q
  driver         = "local"
  labels = {
    tfacc = %q
  }
}

resource "arcane_volume_backup" "test" {
  environment_id = %q
  volume_name    = arcane_volume.test.name
}

resource "arcane_container" "test" {
  environment_id  = %q
  name            = %q
  image           = "alpine:latest"
  command         = ["sh", "-c", "sleep 60"]
  networks        = [arcane_network.test.name]
  volumes         = ["${arcane_volume.test.name}:/data"]
  force_delete    = true
  remove_volumes  = false
  labels = {
    tfacc = %q
  }
}

resource "arcane_vulnerability_ignore" "test" {
  environment_id      = %q
  image_id            = "sha256:%s"
  vulnerability_id    = "CVE-2099-000%s"
  pkg_name            = "tfacc"
  installed_version   = "1.%s.0"
  reason              = "Terraform acceptance %s"
  created_by          = "terraform-provider-arcane"
}

resource "arcane_gitops_sync" "test" {
  environment_id         = %q
  name                   = %q
  repository_id          = arcane_git_repository.test.id
  branch                 = "master"
  compose_path           = "nginx-flask-mysql/compose.yaml"
  project_name           = %q
  auto_sync              = false
  sync_interval          = 3600
  sync_directory         = false
  target_type            = "project"
  start_project          = false
  max_sync_binary_size   = 1048576
  max_sync_files         = 100
  max_sync_total_size    = 5242880
  environment_variables = {
    TFACC_SUFFIX = %q
  }
}
`,
		testAccEndpoint(),
		testAccAPIKey(),
		envName,
		testAccName("user"),
		suffix,
		testAccRunID(),
		suffix,
		testAccName("api-key-"+suffix),
		suffix,
		suffix,
		suffix,
		testAccName("git-repository-"+suffix),
		suffix,
		testAccName("template-"+suffix),
		suffix,
		suffix,
		suffix,
		testAccName("template-registry-"+suffix),
		suffix,
		testAccEnvironmentID(),
		map[string]string{"1": "light", "2": "dark"}[suffix],
		map[string]string{"1": "0 */5 * * * *", "2": "0 */10 * * * *"}[suffix],
		testAccEnvironmentID(),
		map[string]string{"1": "0 */5 * * * *", "2": "0 */10 * * * *"}[suffix],
		testAccEnvironmentID(),
		map[string]string{"1": "true", "2": "false"}[suffix],
		webhookURL,
		testAccEnvironmentID(),
		testAccName("project-"+suffix),
		suffix,
		suffix,
		testAccEnvironmentID(),
		testAccName("project-path-"+suffix),
		filepath.Join(testAccFixtureDirPath(), "docker-compose.yml"),
		filepath.Join(testAccFixtureDirPath(), ".env"),
		testAccEnvironmentID(),
		testAccName("network-"+suffix),
		suffix,
		testAccEnvironmentID(),
		testAccName("volume-"+suffix),
		suffix,
		testAccEnvironmentID(),
		testAccEnvironmentID(),
		testAccName("container-"+suffix),
		suffix,
		testAccEnvironmentID(),
		strings.Repeat(suffix, 64),
		suffix,
		suffix,
		suffix,
		testAccEnvironmentID(),
		testAccName("gitops-"+suffix),
		testAccName("gitops-project-"+suffix),
		suffix,
	)
}

// testAccDataSourcesConfig returns data source blocks that read back the
// resources declared by testAccAllResourcesConfig. Each data source references
// its resource's computed attributes (id / environment_id), which defers the
// read to apply and guarantees the resource exists first. Appended to the
// resource config, never used on its own.
func testAccDataSourcesConfig() string {
	return `
data "arcane_api_key" "test" {
  id = arcane_api_key.test.id
}

data "arcane_container" "test" {
  environment_id = arcane_container.test.environment_id
  id             = arcane_container.test.id
}

data "arcane_container_registry" "test" {
  id = arcane_container_registry.test.id
}

data "arcane_environment" "test" {
  id = arcane_environment.test.id
}

data "arcane_gitops_sync" "test" {
  environment_id = arcane_gitops_sync.test.environment_id
  id             = arcane_gitops_sync.test.id
}

data "arcane_git_repository" "test" {
  id = arcane_git_repository.test.id
}

data "arcane_job_schedules" "test" {
  environment_id = arcane_job_schedules.test.environment_id
}

data "arcane_network" "test" {
  environment_id = arcane_network.test.environment_id
  id             = arcane_network.test.id
}

data "arcane_notification" "test" {
  environment_id = arcane_notification.test.environment_id
  provider_name  = arcane_notification.test.provider_name
}

data "arcane_project" "test" {
  environment_id = arcane_project.test.environment_id
  id             = arcane_project.test.id
}

data "arcane_project_path" "test" {
  environment_id = arcane_project_path.test.environment_id
  id             = arcane_project_path.test.id
}

data "arcane_settings" "test" {
  environment_id = arcane_settings.test.environment_id
}

data "arcane_template" "test" {
  id = arcane_template.test.id
}

data "arcane_template_registry" "test" {
  id = arcane_template_registry.test.id
}

data "arcane_user" "test" {
  id = arcane_user.test.id
}

data "arcane_volume" "test" {
  environment_id = arcane_volume.test.environment_id
  id             = arcane_volume.test.id
}
`
}

// TestAccArcaneProvider_rbac exercises the RBAC surface: custom roles, the role
// and permission-manifest data sources, OIDC group→role mappings, and workload
// identity federated credentials. Step 1 creates everything; step 2 updates each
// resource (rename/permission change, claim change, disable + ttl change).
func TestAccArcaneProvider_rbac(t *testing.T) {
	roleName := testAccName("role")
	fedName := testAccName("fedcred")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccRBACConfig(roleName, fedName, "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					logCheck(t, "arcane_role",
						resource.TestCheckResourceAttrSet("arcane_role.test", "id"),
						resource.TestCheckResourceAttr("arcane_role.test", "name", roleName),
						resource.TestCheckResourceAttr("arcane_role.test", "built_in", "false"),
						resource.TestCheckResourceAttr("arcane_role.test", "permissions.#", "2"),
						resource.TestCheckResourceAttr("arcane_role.test", "description", "RBAC acceptance role 1"),
					),
					logCheck(t, "data.arcane_role",
						resource.TestCheckResourceAttrPair("data.arcane_role.by_name", "id", "arcane_role.test", "id"),
						resource.TestCheckResourceAttr("data.arcane_role.by_name", "built_in", "false"),
					),
					logCheck(t, "data.arcane_role_permissions",
						resource.TestCheckResourceAttrSet("data.arcane_role_permissions.all", "all_permissions.#"),
						resource.TestCheckResourceAttrSet("data.arcane_role_permissions.all", "resources.#"),
					),
					logCheck(t, "arcane_oidc_role_mapping",
						resource.TestCheckResourceAttrSet("arcane_oidc_role_mapping.test", "id"),
						resource.TestCheckResourceAttr("arcane_oidc_role_mapping.test", "claim_value", "tfacc-group-1"),
						resource.TestCheckResourceAttrPair("arcane_oidc_role_mapping.test", "role_id", "arcane_role.test", "id"),
						resource.TestCheckResourceAttr("arcane_oidc_role_mapping.test", "source", "manual"),
					),
					logCheck(t, "arcane_federated_credential",
						resource.TestCheckResourceAttrSet("arcane_federated_credential.test", "id"),
						resource.TestCheckResourceAttr("arcane_federated_credential.test", "name", fedName),
						resource.TestCheckResourceAttr("arcane_federated_credential.test", "enabled", "true"),
						resource.TestCheckResourceAttr("arcane_federated_credential.test", "match_type", "glob"),
						resource.TestCheckResourceAttrPair("arcane_federated_credential.test", "role_id", "arcane_role.test", "id"),
						resource.TestCheckResourceAttrSet("arcane_federated_credential.test", "identity_user_id"),
					),
				),
			},
			{
				Config: testAccRBACConfig(roleName, fedName, "2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					logCheck(t, "arcane_role",
						resource.TestCheckResourceAttr("arcane_role.test", "permissions.#", "3"),
						resource.TestCheckResourceAttr("arcane_role.test", "description", "RBAC acceptance role 2"),
					),
					logCheck(t, "arcane_oidc_role_mapping",
						resource.TestCheckResourceAttr("arcane_oidc_role_mapping.test", "claim_value", "tfacc-group-2"),
					),
					logCheck(t, "arcane_federated_credential",
						resource.TestCheckResourceAttr("arcane_federated_credential.test", "enabled", "false"),
						resource.TestCheckResourceAttr("arcane_federated_credential.test", "token_ttl_seconds", "600"),
					),
				),
			},
		},
	})
}

func testAccRBACConfig(roleName, fedName, suffix string) string {
	permissions := `["containers:list", "projects:read"]`
	description := "RBAC acceptance role 1"
	claim := "tfacc-group-1"
	fedEnabled := "true"
	fedTTL := ""
	if suffix == "2" {
		permissions = `["containers:list", "projects:read", "containers:start"]`
		description = "RBAC acceptance role 2"
		claim = "tfacc-group-2"
		fedEnabled = "false"
		fedTTL = "  token_ttl_seconds = 600\n"
	}

	return fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}

resource "arcane_role" "test" {
  name        = %q
  description = %q
  permissions = %s
}

data "arcane_role" "by_name" {
  name = arcane_role.test.name
}

data "arcane_role_permissions" "all" {}

resource "arcane_oidc_role_mapping" "test" {
  claim_value = %q
  role_id     = arcane_role.test.id
}

resource "arcane_federated_credential" "test" {
  name          = %q
  enabled       = %s
  issuer_url    = "https://issuer.example.test"
  audiences     = ["arcane"]
  subject_match = "repo:example/app:*"
  match_type    = "glob"
  role_id       = arcane_role.test.id
%s}
`, testAccEndpoint(), testAccAPIKey(), roleName, description, permissions, claim, fedName, fedEnabled, fedTTL)
}
