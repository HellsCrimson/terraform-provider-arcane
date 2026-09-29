package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Every secret in these configurations contains this marker, so a single scan
// of the whole state proves none of them was persisted, whatever attribute (or
// nested value) it could have leaked into.
const writeOnlySecretMarker = "wosecret"

// writeOnlyConfig parameterises testAccWriteOnlyConfig.
type writeOnlyConfig struct {
	name string
	// secret is appended to every secret value; changing it without bumping
	// version must not reach Arcane.
	secret  string
	version int
	// dropPreDeployEnv removes pre_deploy_env_wo from the configuration (its
	// version stays), which clears it on the server once the version changes.
	dropPreDeployEnv bool
}

func (c writeOnlyConfig) password() string {
	return "Wo-Pass-" + writeOnlySecretMarker + c.secret + "!x9"
}

func (c writeOnlyConfig) awsAccessKeyID() string {
	return "AKIA" + writeOnlySecretMarker + c.secret
}

func (c writeOnlyConfig) preDeployEnv() string {
	return "SECRET_KEY=" + writeOnlySecretMarker + c.secret
}

// TestAccArcaneWriteOnly_notInState covers every *_wo attribute that runs
// against the default test environment (arcane_swarm_secret needs swarm mode
// and has its own test):
//
//  1. Create: the values reach Arcane, yet the state holds none of them.
//  2. Changing only the secret values plans nothing, because Terraform keeps no
//     copy to diff, and Arcane keeps the old values.
//  3. Bumping the *_wo_version attributes updates in place and sends the new
//     values, still without persisting them.
//  4. Removing pre_deploy_env_wo and bumping its version clears it on the
//     server.
func TestAccArcaneWriteOnly_notInState(t *testing.T) {
	name := testAccName("wo")
	v1 := writeOnlyConfig{name: name, secret: "1", version: 1}
	v1Changed := writeOnlyConfig{name: name, secret: "2", version: 1}
	v2 := writeOnlyConfig{name: name, secret: "2", version: 2}
	v3 := writeOnlyConfig{name: name, secret: "2", version: 3, dropPreDeployEnv: true}

	var expectUpdates []plancheck.PlanCheck
	for addr := range writeOnlyAttrs {
		expectUpdates = append(expectUpdates, plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate))
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccWriteOnlyConfig(v1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNoSecretInState(writeOnlySecretMarker),
					testAccCheckWriteOnlyAttrsNull(),
					resource.TestCheckResourceAttr("arcane_user.wo", "password_wo_version", "1"),
					resource.TestCheckResourceAttr("arcane_git_repository.wo_token", "token_wo_version", "1"),
					resource.TestCheckResourceAttr("arcane_container_registry.wo", "token_wo_version", "1"),
					resource.TestCheckResourceAttr("arcane_settings.wo", "oidc_client_secret_wo_version", "1"),
					resource.TestCheckResourceAttr("arcane_gitops_sync.wo", "pre_deploy_env_wo_version", "1"),
					// The values did reach Arcane.
					testAccCheckUserLogin(name+"-user", v1.password()),
					testAccCheckGitRepositoryCredentials("arcane_git_repository.wo_token", true, false),
					testAccCheckGitRepositoryCredentials("arcane_git_repository.wo_ssh", false, true),
					testAccCheckRegistryAWSAccessKeyID("arcane_container_registry.wo_ecr", v1.awsAccessKeyID()),
					testAccCheckGitOpsSyncPreDeployEnv("arcane_gitops_sync.wo", v1.preDeployEnv()),
				),
			},
			{
				Config: testAccWriteOnlyConfig(v1Changed),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				// No login check here: Arcane rate-limits logins, and the two
				// checks below already prove nothing was sent.
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckRegistryAWSAccessKeyID("arcane_container_registry.wo_ecr", v1.awsAccessKeyID()),
					testAccCheckGitOpsSyncPreDeployEnv("arcane_gitops_sync.wo", v1.preDeployEnv()),
				),
			},
			{
				Config: testAccWriteOnlyConfig(v2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: expectUpdates,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNoSecretInState(writeOnlySecretMarker),
					testAccCheckWriteOnlyAttrsNull(),
					resource.TestCheckResourceAttr("arcane_user.wo", "password_wo_version", "2"),
					resource.TestCheckResourceAttr("arcane_environment.wo", "access_token_wo_version", "2"),
					testAccCheckUserLogin(name+"-user", v2.password()),
					testAccCheckRegistryAWSAccessKeyID("arcane_container_registry.wo_ecr", v2.awsAccessKeyID()),
					testAccCheckGitOpsSyncPreDeployEnv("arcane_gitops_sync.wo", v2.preDeployEnv()),
				),
			},
			{
				Config: testAccWriteOnlyConfig(v3),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNoSecretInState(writeOnlySecretMarker),
					resource.TestCheckResourceAttr("arcane_gitops_sync.wo", "pre_deploy_env_wo_version", "3"),
					testAccCheckGitOpsSyncPreDeployEnv("arcane_gitops_sync.wo", ""),
				),
			},
		},
	})
}

// writeOnlyAttrs lists the secret attributes, stored and write-only, of the
// resources in testAccWriteOnlyConfig. None of them may be in state.
var writeOnlyAttrs = map[string][]string{
	"arcane_user.wo":                   {"password", "password_wo"},
	"arcane_git_repository.wo_token":   {"token", "token_wo", "ssh_key", "ssh_key_wo"},
	"arcane_git_repository.wo_ssh":     {"token", "token_wo", "ssh_key", "ssh_key_wo"},
	"arcane_container_registry.wo":     {"token", "token_wo", "aws_access_key_id", "aws_access_key_id_wo", "aws_secret_access_key", "aws_secret_access_key_wo"},
	"arcane_container_registry.wo_ecr": {"token", "token_wo", "aws_access_key_id", "aws_access_key_id_wo", "aws_secret_access_key", "aws_secret_access_key_wo"},
	"arcane_environment.wo":            {"access_token", "access_token_wo"},
	"arcane_settings.wo":               {"trivy_server_token", "trivy_server_token_wo", "oidc_client_secret", "oidc_client_secret_wo", "depot_token", "depot_token_wo"},
	"arcane_gitops_sync.wo":            {"pre_deploy_env", "pre_deploy_env_wo"},
}

func testAccCheckWriteOnlyAttrsNull() resource.TestCheckFunc {
	var checks []resource.TestCheckFunc
	for addr, attrs := range writeOnlyAttrs {
		for _, attr := range attrs {
			checks = append(checks, resource.TestCheckNoResourceAttr(addr, attr))
		}
	}
	return resource.ComposeAggregateTestCheckFunc(checks...)
}

// testAccCheckNoSecretInState fails when any attribute of any resource in the
// state contains marker.
func testAccCheckNoSecretInState(marker string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for addr, rs := range s.RootModule().Resources {
			if rs.Primary == nil {
				continue
			}
			for k, v := range rs.Primary.Attributes {
				if strings.Contains(v, marker) {
					return fmt.Errorf("%s.%s holds a secret in state", addr, k)
				}
			}
		}
		return nil
	}
}

// testAccCheckUserLogin proves a password reached Arcane by logging in with
// it. Arcane rate-limits logins (429 with Retry-After), so a throttled attempt
// waits and retries a few times rather than failing the test.
func testAccCheckUserLogin(username, password string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		body, _ := json.Marshal(map[string]string{"username": username, "password": password})
		url := strings.TrimSuffix(testAccEndpoint(), "/") + "/auth/login"
		client := &http.Client{Timeout: 30 * time.Second}
		for attempt := 0; ; attempt++ {
			res, err := client.Post(url, "application/json", bytes.NewReader(body))
			if err != nil {
				return fmt.Errorf("login as %s: %w", username, err)
			}
			res.Body.Close()
			if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				wait := 60 * time.Second
				if secs, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil {
					wait = time.Duration(secs) * time.Second
				}
				time.Sleep(wait)
				continue
			}
			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("login as %s with the configured password: unexpected status %s", username, res.Status)
			}
			return nil
		}
	}
}

func testAccResourceID(s *terraform.State, addr string) (string, error) {
	rs, ok := s.RootModule().Resources[addr]
	if !ok || rs.Primary == nil {
		return "", fmt.Errorf("%s not found in state", addr)
	}
	return rs.Primary.ID, nil
}

func testAccCheckGitRepositoryCredentials(addr string, wantToken, wantSSHKey bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccResourceID(s, addr)
		if err != nil {
			return err
		}
		repo, err := sdkclient.NewClient(testAccEndpoint(), testAccAPIKey()).GetGitRepository(context.Background(), id)
		if err != nil {
			return err
		}
		if repo.HasToken != wantToken || repo.HasSSHKey != wantSSHKey {
			return fmt.Errorf("%s: Arcane reports hasToken=%t hasSshKey=%t, want %t/%t", addr, repo.HasToken, repo.HasSSHKey, wantToken, wantSSHKey)
		}
		return nil
	}
}

func testAccCheckRegistryAWSAccessKeyID(addr, want string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testAccResourceID(s, addr)
		if err != nil {
			return err
		}
		reg, err := sdkclient.NewClient(testAccEndpoint(), testAccAPIKey()).GetContainerRegistry(context.Background(), id)
		if err != nil {
			return err
		}
		if reg.AWSAccessKeyID != want {
			return fmt.Errorf("%s: Arcane has awsAccessKeyId %q, want %q", addr, reg.AWSAccessKeyID, want)
		}
		return nil
	}
}

func testAccCheckGitOpsSyncPreDeployEnv(addr, want string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[addr]
		if !ok || rs.Primary == nil {
			return fmt.Errorf("%s not found in state", addr)
		}
		sync, err := sdkclient.NewClient(testAccEndpoint(), testAccAPIKey()).GetGitOpsSync(context.Background(), rs.Primary.Attributes["environment_id"], rs.Primary.ID)
		if err != nil {
			return err
		}
		got := ""
		if sync.PreDeployEnv != nil {
			got = *sync.PreDeployEnv
		}
		if got != want {
			return fmt.Errorf("%s: Arcane has preDeployEnv %q, want %q", addr, got, want)
		}
		return nil
	}
}

func testAccWriteOnlyConfig(c writeOnlyConfig) string {
	s := writeOnlySecretMarker + c.secret
	preDeployEnv := fmt.Sprintf("pre_deploy_env_wo         = %q\n  ", c.preDeployEnv())
	if c.dropPreDeployEnv {
		preDeployEnv = ""
	}

	return fmt.Sprintf(`
provider "arcane" {
  endpoint     = %[1]q
  api_key      = %[2]q
  http_timeout = "180s"
}

resource "arcane_user" "wo" {
  username            = "%[4]s-user"
  password_wo         = %[6]q
  password_wo_version = %[5]d
}

resource "arcane_git_repository" "wo_token" {
  name             = "%[4]s-token"
  url              = "https://github.com/docker/awesome-compose.git"
  auth_type        = "token"
  username         = "tfacc"
  token_wo         = "git-token-%[7]s"
  token_wo_version = %[5]d
}

resource "arcane_git_repository" "wo_ssh" {
  name               = "%[4]s-ssh"
  url                = "git@github.com:docker/awesome-compose.git"
  auth_type          = "ssh"
  ssh_key_wo         = "ssh-key-%[7]s"
  ssh_key_wo_version = %[5]d
}

resource "arcane_container_registry" "wo" {
  url              = "https://%[4]s.example.test"
  username         = "tfacc"
  token_wo         = "registry-token-%[7]s"
  token_wo_version = %[5]d
}

# Arcane only keeps AWS credentials on ECR registries. It also drops the
# username there, hence the empty one.
resource "arcane_container_registry" "wo_ecr" {
  url                              = "https://123456789012.dkr.ecr.eu-west-1.amazonaws.com"
  registry_type                    = "ecr"
  username                         = ""
  token_wo                         = "registry-token-%[7]s"
  token_wo_version                 = %[5]d
  aws_access_key_id_wo             = %[8]q
  aws_access_key_id_wo_version     = %[5]d
  aws_secret_access_key_wo         = "aws-secret-%[7]s"
  aws_secret_access_key_wo_version = %[5]d
  aws_region                       = "eu-west-1"
}

resource "arcane_environment" "wo" {
  name                    = "%[4]s-env"
  api_url                 = "http://localhost:3552"
  enabled                 = true
  access_token_wo         = "env-token-%[7]s"
  access_token_wo_version = %[5]d
}

resource "arcane_settings" "wo" {
  environment_id                = %[3]q
  lifecycle_enabled             = "true"
  trivy_server_token_wo         = "trivy-%[7]s"
  trivy_server_token_wo_version = %[5]d
  oidc_client_secret_wo         = "oidc-%[7]s"
  oidc_client_secret_wo_version = %[5]d
  depot_token_wo                = "depot-%[7]s"
  depot_token_wo_version        = %[5]d
}

resource "arcane_git_repository" "wo_public" {
  name      = "%[4]s-public"
  url       = "https://github.com/docker/awesome-compose.git"
  auth_type = "none"
  enabled   = true
}

resource "arcane_gitops_sync" "wo" {
  depends_on = [arcane_settings.wo]

  environment_id = %[3]q
  name           = "%[4]s-sync"
  repository_id  = arcane_git_repository.wo_public.id
  branch         = "master"
  compose_path   = "nginx-flask-mysql/compose.yaml"
  project_name   = "%[4]s-sync"
  auto_sync      = false
  sync_directory = true
  target_type    = "project"
  start_project  = false

  pre_deploy_script_path    = "pre-deploy.sh"
  pre_deploy_runner_image   = "alpine:3"
  %[9]spre_deploy_env_wo_version = %[5]d
}
`, testAccEndpoint(), testAccAPIKey(), testAccEnvironmentID(), c.name, c.version, c.password(), s, c.awsAccessKeyID(), preDeployEnv)
}

// TestAccArcaneWriteOnly_migrateFromStored covers moving from the deprecated
// stored attributes to their *_wo counterparts:
//
//  1. The stored attributes still work, and still persist the secret.
//  2. Switching to *_wo (with a first *_wo_version) is an in-place update that
//     drops the secret from state. The user's password changes in the same
//     step and the login proves the write-only value was sent.
func TestAccArcaneWriteOnly_migrateFromStored(t *testing.T) {
	name := testAccName("wo-migrate")
	oldPassword := "Wo-Pass-" + writeOnlySecretMarker + "old!x9"
	newPassword := "Wo-Pass-" + writeOnlySecretMarker + "new!x9"

	var expectUpdates []plancheck.PlanCheck
	for _, addr := range []string{"arcane_user.wo", "arcane_git_repository.wo", "arcane_container_registry.wo", "arcane_environment.wo"} {
		expectUpdates = append(expectUpdates, plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate))
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccWriteOnlyMigrateConfig(name, oldPassword, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					// The precondition: the deprecated attributes persist.
					resource.TestCheckResourceAttr("arcane_user.wo", "password", oldPassword),
					resource.TestCheckResourceAttr("arcane_git_repository.wo", "token", "git-token-"+writeOnlySecretMarker),
					resource.TestCheckResourceAttr("arcane_container_registry.wo", "token", "registry-token-"+writeOnlySecretMarker),
					resource.TestCheckResourceAttr("arcane_environment.wo", "access_token", "env-token-"+writeOnlySecretMarker),
					testAccCheckUserLogin(name+"-user", oldPassword),
				),
			},
			{
				Config: testAccWriteOnlyMigrateConfig(name, newPassword, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: expectUpdates,
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNoSecretInState(writeOnlySecretMarker),
					resource.TestCheckNoResourceAttr("arcane_user.wo", "password"),
					resource.TestCheckNoResourceAttr("arcane_git_repository.wo", "token"),
					resource.TestCheckNoResourceAttr("arcane_container_registry.wo", "token"),
					resource.TestCheckNoResourceAttr("arcane_environment.wo", "access_token"),
					testAccCheckUserLogin(name+"-user", newPassword),
					testAccCheckGitRepositoryCredentials("arcane_git_repository.wo", true, false),
				),
			},
		},
	})
}

func testAccWriteOnlyMigrateConfig(name, password string, writeOnly bool) string {
	suffix := ""
	versionLine := func(string) string { return "" }
	if writeOnly {
		suffix = "_wo"
		versionLine = func(attr string) string { return "\n  " + attr + "_wo_version = 1" }
	}

	return fmt.Sprintf(`
provider "arcane" {
  endpoint     = %[1]q
  api_key      = %[2]q
  http_timeout = "180s"
}

resource "arcane_user" "wo" {
  username = "%[3]s-user"
  password%[5]s = %[4]q%[6]s
}

resource "arcane_git_repository" "wo" {
  name      = "%[3]s-repo"
  url       = "https://github.com/docker/awesome-compose.git"
  auth_type = "token"
  username  = "tfacc"
  token%[5]s = "git-token-%[9]s"%[7]s
}

resource "arcane_container_registry" "wo" {
  url      = "https://%[3]s.example.test"
  username = "tfacc"
  token%[5]s = "registry-token-%[9]s"%[7]s
}

resource "arcane_environment" "wo" {
  name    = "%[3]s-env"
  api_url = "http://localhost:3552"
  enabled = true
  access_token%[5]s = "env-token-%[9]s"%[8]s
}
`, testAccEndpoint(), testAccAPIKey(), name, password, suffix,
		versionLine("password"), versionLine("token"), versionLine("access_token"), writeOnlySecretMarker)
}

// TestAccArcaneWriteOnly_validation pins the plan-time rules tying each secret
// to its counterpart: the stored and write-only flavours are exclusive, a
// required secret needs one of them, and a *_wo value needs its version.
func TestAccArcaneWriteOnly_validation(t *testing.T) {
	provider := fmt.Sprintf(`
provider "arcane" {
  endpoint = %q
  api_key  = %q
}
`, testAccEndpoint(), testAccAPIKey())

	cases := []struct {
		config string
		want   string
	}{
		{`
resource "arcane_git_repository" "v" {
  name             = "v"
  url              = "https://example.test/v.git"
  auth_type        = "token"
  token            = "a"
  token_wo         = "b"
  token_wo_version = 1
}`, `(?s)token_wo.*cannot be specified when.*token`},
		{`
resource "arcane_user" "v" {
  username = "v"
}`, `(?s)No attribute specified when one \(and only one\) of.*password_wo`},
		{`
resource "arcane_user" "v" {
  username    = "v"
  password_wo = "long-enough-1!"
}`, `(?s)password_wo_version.*must be specified when.*password_wo`},
	}
	var steps []resource.TestStep
	for _, c := range cases {
		steps = append(steps, resource.TestStep{
			Config:      provider + c.config,
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(c.want),
		})
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps:                    steps,
	})
}

// TestAccArcaneSwarmSecret_writeOnlyData checks the swarm secret's data_wo
// stays out of state, that bumping data_wo_version replaces the (immutable)
// secret, and that moving from the deprecated data to data_wo keeps the
// secret. It needs an environment whose Docker engine is a swarm manager, so
// it skips when Arcane reports swarm mode as disabled.
func TestAccArcaneSwarmSecret_writeOnlyData(t *testing.T) {
	testAccSkipUnlessSwarm(t)
	name := testAccName("wo-secret")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccSwarmSecretStoredConfig(name),
				Check:  resource.TestCheckResourceAttr("arcane_swarm_secret.wo", "data", writeOnlySecretMarker+"1"),
			},
			{
				// data -> data_wo with the same value: no replacement.
				Config: testAccSwarmSecretWriteOnlyConfig(name, "1", 1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("arcane_swarm_secret.wo", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNoSecretInState(writeOnlySecretMarker),
					resource.TestCheckNoResourceAttr("arcane_swarm_secret.wo", "data"),
					resource.TestCheckResourceAttr("arcane_swarm_secret.wo", "data_wo_version", "1"),
				),
			},
			{
				Config: testAccSwarmSecretWriteOnlyConfig(name, "2", 1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: testAccSwarmSecretWriteOnlyConfig(name, "2", 2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("arcane_swarm_secret.wo", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNoSecretInState(writeOnlySecretMarker),
					resource.TestCheckResourceAttr("arcane_swarm_secret.wo", "data_wo_version", "2"),
				),
			},
		},
	})
}

func testAccSwarmSecretStoredConfig(name string) string {
	return fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}

resource "arcane_swarm_secret" "wo" {
  environment_id = %q
  name           = %q
  data           = "%s1"
}
`, testAccEndpoint(), testAccAPIKey(), testAccEnvironmentID(), name, writeOnlySecretMarker)
}

func testAccSwarmSecretWriteOnlyConfig(name, secret string, version int) string {
	return fmt.Sprintf(`
provider "arcane" {
  endpoint     = %q
  api_key      = %q
  http_timeout = "180s"
}

resource "arcane_swarm_secret" "wo" {
  environment_id  = %q
  name            = %q
  data_wo         = "%s%s"
  data_wo_version = %d
}
`, testAccEndpoint(), testAccAPIKey(), testAccEnvironmentID(), name, writeOnlySecretMarker, secret, version)
}

// testAccSkipUnlessSwarm skips a test when the environment under test is not a
// swarm manager (Arcane answers the swarm endpoints with 409 Conflict).
func testAccSkipUnlessSwarm(t *testing.T) {
	t.Helper()

	// Runs before resource.Test, so it has to honour the acceptance-test gate
	// itself: without TF_ACC there is no server to ask.
	if os.Getenv("TF_ACC") == "" {
		return
	}
	url := strings.TrimSuffix(testAccEndpoint(), "/") + "/environments/" + testAccEnvironmentID() + "/swarm/secrets"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("building swarm probe request: %s", err)
	}
	req.Header.Set("X-API-Key", testAccAPIKey())
	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("probing swarm mode: %s", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusConflict {
		t.Skip("swarm mode is not enabled on the test environment")
	}
}
