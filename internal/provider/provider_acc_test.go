package provider

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	testEndpoint = "http://localhost:3552/api"
	testAPIKey   = "arc_a54fe1040057252a19b34d72008395141a04de7731a28d6f7359baa4923b2f6a"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"arcane": providerserver.NewProtocol6WithError(New("test")()),
}

func TestAccArcaneProvider_allResources(t *testing.T) {
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
				Config: testAccAllResourcesConfig("1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arcane_environment.test", "id"),
					resource.TestCheckResourceAttr("arcane_environment.test", "name", testAccName("env-1")),
					resource.TestCheckResourceAttrSet("arcane_user.test", "id"),
					resource.TestCheckResourceAttr("arcane_user.test", "display_name", "Terraform Acceptance User 1"),
					resource.TestCheckResourceAttrSet("arcane_api_key.test", "id"),
					resource.TestCheckResourceAttrSet("arcane_container_registry.test", "id"),
					resource.TestCheckResourceAttrSet("arcane_git_repository.test", "id"),
					resource.TestCheckResourceAttrSet("arcane_template.test", "id"),
					resource.TestCheckResourceAttrSet("arcane_template_registry.test", "id"),
					resource.TestCheckResourceAttrSet("arcane_project.test", "id"),
					resource.TestCheckResourceAttrSet("arcane_project_path.test", "id"),
					resource.TestCheckResourceAttrSet("arcane_container.test", "id"),
					resource.TestCheckResourceAttrSet("arcane_network.test", "id"),
					resource.TestCheckResourceAttrSet("arcane_volume.test", "id"),
					resource.TestCheckResourceAttrSet("arcane_volume_backup.test", "id"),
					resource.TestCheckResourceAttrSet("arcane_vulnerability_ignore.test", "id"),
					resource.TestCheckResourceAttr("arcane_job_schedules.test", "polling_interval", "0 */5 * * * *"),
					resource.TestCheckResourceAttr("arcane_settings.test", "application_theme", "light"),
				),
			},
			{
				PreConfig: func() {
					writeProjectPathFixtures(t, "2")
				},
				Config: testAccAllResourcesConfig("2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_environment.test", "name", testAccName("env-2")),
					resource.TestCheckResourceAttr("arcane_user.test", "display_name", "Terraform Acceptance User 2"),
					resource.TestCheckResourceAttr("arcane_api_key.test", "name", testAccName("api-key-2")),
					resource.TestCheckResourceAttr("arcane_container_registry.test", "description", "Terraform acceptance registry 2"),
					resource.TestCheckResourceAttr("arcane_git_repository.test", "description", "Terraform acceptance repository 2"),
					resource.TestCheckResourceAttr("arcane_template.test", "description", "Terraform acceptance template 2"),
					resource.TestCheckResourceAttr("arcane_template_registry.test", "description", "Terraform acceptance template registry 2"),
					resource.TestCheckResourceAttr("arcane_project.test", "name", testAccName("project-2")),
					resource.TestCheckResourceAttr("arcane_project_path.test", "name", testAccName("project-path-2")),
					resource.TestCheckResourceAttr("arcane_container.test", "name", testAccName("container-2")),
					resource.TestCheckResourceAttr("arcane_network.test", "name", testAccName("network-2")),
					resource.TestCheckResourceAttr("arcane_volume.test", "name", testAccName("volume-2")),
					resource.TestCheckResourceAttr("arcane_job_schedules.test", "polling_interval", "0 */10 * * * *"),
					resource.TestCheckResourceAttr("arcane_settings.test", "application_theme", "dark"),
					resource.TestCheckResourceAttr("arcane_vulnerability_ignore.test", "reason", "Terraform acceptance 2"),
				),
			},
		},
	})
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

func testAccAllResourcesConfig(suffix string) string {
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
  roles        = ["user"]
}

resource "arcane_api_key" "test" {
  name        = %q
  description = "Terraform acceptance API key %s"
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
  environment_id = arcane_environment.test.id
  provider_name  = "discord"
  enabled        = %s
  config = {
    avatarUrl = ""
    events = {
      container_update    = true
      image_update        = false
      prune_report        = true
      vulnerability_found = false
    }
    token     = "token-%s"
    username  = "tfacc"
    webhookId = "webhook-%s"
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
		map[string]string{"1": "true", "2": "false"}[suffix],
		suffix,
		suffix,
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
