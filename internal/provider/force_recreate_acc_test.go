package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"terraform-provider-arcane/internal/sdkclient"
)

// The force_recreate acceptance tests use the ID of the project's container as
// the observable side effect: compose keeps a container whose configuration did
// not change, so a redeploy only replaces it when force_recreate is set.

// TestAccArcaneProject_forceRecreate redeploys an unchanged project with
// redeploy_trigger = "always": the container survives the redeploy until
// force_recreate is set, and is replaced on every redeploy after that.
func TestAccArcaneProject_forceRecreate(t *testing.T) {
	name := testAccName("force-recreate")
	var containerID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config:             testAccProjectRedeployTriggerConfig(name, `redeploy_trigger = "always"`, "v1", ""),
				ExpectNonEmptyPlan: true,
				Check:              testAccCaptureProjectContainer(name, &containerID),
			},
			{
				// Redeployed, but nothing changed: compose keeps the container.
				Config:             testAccProjectRedeployTriggerConfig(name, `redeploy_trigger = "always"`, "v1", ""),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arcane_project.test", "last_redeploy"),
					testAccCheckProjectContainerKept(name, &containerID),
				),
			},
			{
				Config:             testAccProjectRedeployTriggerConfig(name, `redeploy_trigger = "always"`, "v1", "\n  force_recreate = true"),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project.test", "force_recreate", "true"),
					testAccCheckProjectContainerReplaced(name, &containerID),
				),
			},
			{
				Config:             testAccProjectRedeployTriggerConfig(name, `redeploy_trigger = "always"`, "v1", "\n  force_recreate = true"),
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckProjectContainerReplaced(name, &containerID),
			},
		},
	})
}

// TestAccArcaneProjectPath_forceRecreate covers the same on
// arcane_project_path: with force_recreate, a redeploy of unchanged files
// replaces the container.
func TestAccArcaneProjectPath_forceRecreate(t *testing.T) {
	name := testAccName("force-recreate-path")
	var containerID string
	config := strings.Replace(testAccProjectPathRedeployTriggerConfig(name, "always"),
		"pull_on_update    = false", "pull_on_update    = false\n  force_recreate    = true", 1)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				PreConfig:          func() { writeProjectPathFixtures(t, "1") },
				Config:             config,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("arcane_project_path.test", "force_recreate", "true"),
					testAccCaptureProjectContainer(name, &containerID),
				),
			},
			{
				PreConfig:          func() { writeProjectPathFixtures(t, "1") },
				Config:             config,
				ExpectNonEmptyPlan: true,
				Check:              testAccCheckProjectContainerReplaced(name, &containerID),
			},
		},
	})
}

// testAccProjectContainerID returns the ID of the single container of the
// project's "app" service.
func testAccProjectContainerID(projectName string) (string, error) {
	client := sdkclient.NewClient(testAccEndpoint(), testAccAPIKey())
	containers, err := client.ListContainers(context.Background(), testAccEnvironmentID())
	if err != nil {
		return "", fmt.Errorf("listing containers: %w", err)
	}
	want := "/" + projectName + "-app-1"
	for _, c := range containers {
		for _, n := range c.Names {
			if n == want || n == strings.TrimPrefix(want, "/") {
				return c.ID, nil
			}
		}
	}
	return "", fmt.Errorf("no container named %s", want)
}

func testAccCaptureProjectContainer(projectName string, out *string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		id, err := testAccProjectContainerID(projectName)
		if err != nil {
			return err
		}
		*out = id
		return nil
	}
}

func testAccCheckProjectContainerKept(projectName string, prev *string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		id, err := testAccProjectContainerID(projectName)
		if err != nil {
			return err
		}
		if id != *prev {
			return fmt.Errorf("container was recreated (%s -> %s), want it kept", *prev, id)
		}
		return nil
	}
}

// testAccCheckProjectContainerReplaced also records the new ID, so the next
// step compares against it.
func testAccCheckProjectContainerReplaced(projectName string, prev *string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		id, err := testAccProjectContainerID(projectName)
		if err != nil {
			return err
		}
		if id == *prev {
			return fmt.Errorf("container %s was kept, want it recreated", id)
		}
		*prev = id
		return nil
	}
}
