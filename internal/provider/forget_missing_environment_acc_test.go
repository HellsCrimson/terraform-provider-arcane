package provider

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccArcaneForgetMissingEnvironment is the end-to-end test for the
// forget_missing_environments provider option. It reproduces the real-world
// scenario that motivated the option: a per-environment resource exists in
// state, its environment is deleted/recreated, and a subsequent plan must not
// hard-fail on the manager's "environment not found" error.
//
// Flow:
//  1. Create an agent environment through Terraform (arcane_environment) and
//     start an Arcane agent with its enrollment_token. The agent's Docker
//     backend is the dind daemon from tests/docker-compose.edge.yml.
//  2. Create a project in that environment (provider has
//     forget_missing_environments = true).
//  3. Delete the environment out-of-band (PreConfig of the next step).
//  4. Plan again: the provider's Read hits "environment not found", drops the
//     project from state, and the plan re-plans it as a create.
//
// Requires the docker CLI and the edge stack (dind on the arcane-test network)
// — use `make test-acc-forget`, which sets ARCANE_ACC_AGENT_TEST=1. Skipped
// otherwise (so the normal suite is unaffected).
//
// The agent container is configurable through:
//   - ARCANE_ACC_AGENT_IMAGE        (default ghcr.io/getarcaneapp/agent:v3.0.1)
//   - ARCANE_ACC_AGENT_NETWORK      (default arcane-test)
//   - ARCANE_ACC_AGENT_MANAGER_URL  (default http://arcane:3552)
//   - ARCANE_ACC_AGENT_DOCKER_HOST  (default tcp://dind:2375)
func TestAccArcaneForgetMissingEnvironment(t *testing.T) {
	if os.Getenv("ARCANE_ACC_AGENT_TEST") == "" {
		t.Skip("set ARCANE_ACC_AGENT_TEST=1 and run the edge stack (make test-acc-forget) to run this test")
	}
	testAccPreCheck(t)
	if _, err := exec.LookPath("docker"); err != nil {
		t.Fatalf("the docker CLI is required to start the agent: %s", err)
	}

	client := sdkclient.NewClient(testAccEndpoint(), testAccAPIKey())
	envName := testAccName("forget-env")
	projectName := testAccName("forget-project")
	var envID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1. Create the agent environment and enroll an agent with its token.
			{
				Config: testAccForgetEnvironmentConfig(envName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arcane_environment.agent", "enrollment_token"),
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["arcane_environment.agent"]
						envID = rs.Primary.ID
						startTestAgent(t, envName, rs.Primary.Attributes["enrollment_token"])
						waitAgentEnvironmentReady(t, client, envID)
						return nil
					},
				),
			},
			// 2. Create a project in the agent environment.
			{
				Config: testAccForgetEnvironmentConfig(envName) + testAccForgetProjectConfig(projectName),
				Check:  resource.TestCheckResourceAttrPair("arcane_project.test", "environment_id", "arcane_environment.agent", "id"),
			},
			// 3 + 4. Delete the environment, then plan the project alone: it is
			// forgotten from state on refresh and re-planned as a create.
			{
				PreConfig: func() {
					if err := client.DeleteEnvironment(context.Background(), envID); err != nil {
						t.Fatalf("deleting agent environment out-of-band: %s", err)
					}
				},
				Config:             testAccForgetEnvironmentConfig(envName) + testAccForgetProjectConfig(projectName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("arcane_project.test", plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func testAccEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// startTestAgent runs an Arcane agent that enrolls into the environment with
// token, and removes it when the test ends.
func startTestAgent(t *testing.T, envName, token string) {
	t.Helper()
	name := "arcane-test-agent-" + strings.ToLower(strings.ReplaceAll(envName, "_", "-"))
	args := []string{
		"run", "-d", "--name", name,
		"--network", testAccEnvOrDefault("ARCANE_ACC_AGENT_NETWORK", "arcane-test"),
		"-e", "ARCANE_AGENT=true",
		"-e", "ARCANE_AGENT_TRANSPORT=auto",
		"-e", "ARCANE_AGENT_TOKEN=" + token,
		"-e", "ARCANE_MANAGER_API_URL=" + testAccEnvOrDefault("ARCANE_ACC_AGENT_MANAGER_URL", "http://arcane:3552"),
		"-e", "DOCKER_HOST=" + testAccEnvOrDefault("ARCANE_ACC_AGENT_DOCKER_HOST", "tcp://dind:2375"),
		testAccEnvOrDefault("ARCANE_ACC_AGENT_IMAGE", "ghcr.io/getarcaneapp/agent:v3.0.1"),
	}
	if out, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
		t.Fatalf("starting the agent container: %s: %s", err, out)
	}
	t.Cleanup(func() {
		if t.Failed() {
			if out, err := exec.Command("docker", "logs", "--tail", "50", name).CombinedOutput(); err == nil {
				t.Logf("agent logs:\n%s", out)
			}
		}
		_ = exec.Command("docker", "rm", "-f", name).Run()
	})
}

// waitAgentEnvironmentReady blocks until the agent has enrolled AND its Docker
// backend (the dind daemon) is reachable. ListContainers proxies all the way
// through manager -> agent -> dockerd, so its success guarantees the whole chain
// is up; enrollment alone completes before dind finishes booting, which would
// race the first resource create.
func waitAgentEnvironmentReady(t *testing.T, client *sdkclient.Client, envID string) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(180 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		if _, err := client.ListContainers(ctx, envID); err == nil {
			return
		} else {
			lastErr = err
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("agent environment %s never became ready; the agent failed to enroll or its "+
		"Docker daemon was unreachable (check the arcane-test-agent-* and arcane-test-dind "+
		"containers). last error: %v", envID, lastErr)
}

func testAccForgetEnvironmentConfig(envName string) string {
	return fmt.Sprintf(`
provider "arcane" {
  endpoint                    = %q
  api_key                     = %q
  http_timeout                = "180s"
  forget_missing_environments = true
}

resource "arcane_environment" "agent" {
  name    = %q
  enabled = true
}
`, testAccEndpoint(), testAccAPIKey(), envName)
}

func testAccForgetProjectConfig(name string) string {
	return fmt.Sprintf(`
resource "arcane_project" "test" {
  environment_id   = arcane_environment.agent.id
  name             = %q
  compose_content  = <<YAML
services:
  app:
    image: alpine:latest
    command: ["sh", "-c", "sleep 1"]
YAML
  running          = false
  redeploy_trigger = "never"
  pull_on_update   = false
  remove_files     = true
  remove_volumes   = true
}
`, name)
}
