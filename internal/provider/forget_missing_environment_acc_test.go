package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccArcaneForgetMissingEnvironment is the end-to-end test for the
// forget_missing_environments provider option. It reproduces the real-world
// scenario that motivated the option: a per-environment resource exists in
// state, its environment is deleted/recreated, and a subsequent plan must not
// hard-fail on the manager's "environment not found" proxy error.
//
// Flow:
//  1. Create an edge environment via the API using a pre-shared token; the
//     edge agent from tests/docker-compose.edge.yml polls and auto-pairs.
//  2. Create a project in that environment through Terraform (provider has
//     forget_missing_environments = true).
//  3. Delete the environment out-of-band (PreConfig of the next step).
//  4. Plan again: the provider's Read hits "environment not found", drops the
//     resource from state, and the plan re-plans it as a create.
//
// Requires the edge stack to be running and ARCANE_ACC_EDGE_AGENT_TOKEN set —
// use `make test-acc-forget`. Skipped otherwise (so the normal suite is
// unaffected).
func TestAccArcaneForgetMissingEnvironment(t *testing.T) {
	token := os.Getenv("ARCANE_ACC_EDGE_AGENT_TOKEN")
	if token == "" {
		t.Skip("set ARCANE_ACC_EDGE_AGENT_TOKEN and run the edge stack (make test-acc-forget) to run this test")
	}
	testAccPreCheck(t)

	agentURL := os.Getenv("ARCANE_ACC_EDGE_AGENT_API_URL")
	if agentURL == "" {
		agentURL = "http://arcane-test-edge-agent:3552"
	}

	ctx := context.Background()
	client := sdkclient.NewClient(testAccEndpoint(), testAccAPIKey())

	// 1. Create the edge environment the agent pairs to.
	envName := testAccName("forget-env")
	isEdge, enabled, useAPIKey := true, true, false
	env, err := client.CreateEnvironment(ctx, sdkclient.EnvironmentCreateRequest{
		APIURL:      agentURL,
		Name:        &envName,
		AccessToken: &token,
		Enabled:     &enabled,
		IsEdge:      &isEdge,
		UseAPIKey:   &useAPIKey,
	})
	if err != nil {
		t.Fatalf("creating edge environment: %s", err)
	}
	envID := env.ID

	// Safety net: remove the environment even if the test aborts before the
	// out-of-band delete step.
	envDeleted := false
	t.Cleanup(func() {
		if !envDeleted {
			_ = client.DeleteEnvironment(context.Background(), envID)
		}
	})

	// Wait for the agent to pair — proxied calls succeed once it does.
	waitEdgeEnvironmentReady(t, client, envID)

	projectName := testAccName("forget-project")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 2. Create a project in the edge environment.
			{
				Config: testAccForgetProjectConfig(envID, projectName),
				Check:  resource.TestCheckResourceAttr("arcane_project.test", "environment_id", envID),
			},
			// 3 + 4. Delete the environment, then plan: the resource is forgotten
			// from state on refresh and re-planned as a create.
			{
				PreConfig: func() {
					if err := client.DeleteEnvironment(context.Background(), envID); err != nil {
						t.Fatalf("deleting edge environment out-of-band: %s", err)
					}
					envDeleted = true
				},
				Config:             testAccForgetProjectConfig(envID, projectName),
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

// waitEdgeEnvironmentReady blocks until the edge agent has paired AND its Docker
// backend (the dind daemon) is reachable. ListContainers proxies all the way
// through manager -> agent -> dockerd, so its success guarantees the whole chain
// is up; pairing alone completes before dind finishes booting, which would race
// the first resource create.
func waitEdgeEnvironmentReady(t *testing.T, client *sdkclient.Client, envID string) {
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
	t.Fatalf("edge environment %s never became ready; the agent failed to pair or its "+
		"Docker daemon was unreachable (check the arcane-test-edge-agent and arcane-test-dind "+
		"containers, and that AGENT_TOKEN matches). last error: %v", envID, lastErr)
}

func testAccForgetProjectConfig(envID, name string) string {
	return fmt.Sprintf(`
provider "arcane" {
  endpoint                    = %q
  api_key                     = %q
  http_timeout                = "180s"
  forget_missing_environments = true
}

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
`, testAccEndpoint(), testAccAPIKey(), envID, name)
}
