package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Arcane refuses to rename a project that is not stopped:
//
//	arcane API error: 400 Bad Request: {..."detail":"Failed to update project:
//	project must be stopped before renaming (current status: running)"}
//
// Nothing about a rename hints at that requirement, so Terraform used to plan
// one happily and only discover it halfway through the apply (issue #23). The
// helpers here let both project resources report the requirement during the
// plan, and satisfy it themselves when the configuration asks them to.

// projectStatusStopped is the only project status Arcane will rename in.
const projectStatusStopped = "stopped"

const stopBeforeRenameDescription = "When a rename is planned, stop the project before renaming it and start it again afterwards. Arcane only renames stopped projects, so without this the plan fails when a rename targets a project that is running. The project is down for the duration of the rename."

const renameRequiresStopSummary = "rename requires a stopped project"

// The way out of a rejected rename depends on the resource: the ones that
// manage the project lifecycle can also be told to leave the project down.
const (
	renameRemedyLifecycle = "Either set stop_before_rename = true, which stops the project, renames it and starts it again in a single apply, or set running = false in the same change so the project is stopped before the rename."
	renameRemedyStopOnly  = "Either set stop_before_rename = true, which stops the project, renames it and starts it again in a single apply, or stop the project before applying this change."
)

// projectStopped reports whether a project in this status can be renamed.
func projectStopped(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), projectStatusStopped)
}

// projectPlanStops reports whether the planned lifecycle brings the project
// down on its own. Such a plan can rename after the stop, so it needs no
// stop_before_rename opt-in. Pass a null archived for resources without that
// attribute.
func projectPlanStops(running, archived types.Bool) bool {
	if boolValue(archived) {
		return true
	}
	return !running.IsNull() && !running.IsUnknown() && !running.ValueBool()
}

// renameCheck describes the rename a plan carries, in the terms
// planRenameRequiresStop needs.
type renameCheck struct {
	envID  string
	projID string
	// oldName comes from state, newName from the plan (and may be unknown). An
	// empty oldName means state does not track the name (the GitOps sync only
	// stores a configured project_name), so the API answers for it.
	oldName string
	newName types.String
	// stopBeforeRename and planStops are the two ways the apply ends up
	// renaming a stopped project; either one makes the check moot.
	stopBeforeRename bool
	planStops        bool
	// nameAttr is the attribute the rename is configured through, so the error
	// points at it. Defaults to "name".
	nameAttr string
	// remedy tells the practitioner how to get the rename applied. Defaults to
	// renameRemedyLifecycle.
	remedy string
}

// planRenameRequiresStop fails the plan when it renames a project Arcane would
// refuse to rename, so the practitioner learns about the stop requirement
// before any change is applied rather than from a failed apply.
//
// The status comes from the API rather than from state, because a plan can run
// against state that was never refreshed (terraform plan -refresh=false).
func planRenameRequiresStop(ctx context.Context, client *sdkclient.Client, resp *resource.ModifyPlanResponse, check renameCheck) {
	if client == nil || check.projID == "" {
		return
	}
	if check.newName.IsNull() || check.newName.IsUnknown() {
		return
	}
	newName := check.newName.ValueString()
	if check.oldName != "" && newName == check.oldName {
		return
	}
	if check.stopBeforeRename || check.planStops {
		return
	}

	out, err := client.GetProject(ctx, check.envID, check.projID)
	if err != nil {
		if client.IsResourceGone(err) {
			// The project is gone, so this plan is stale: a refreshed run plans
			// a create, which renames nothing.
			return
		}
		resp.Diagnostics.AddError("failed to check the project status before renaming", err.Error())
		return
	}
	oldName := check.oldName
	if oldName == "" {
		oldName = out.Name
	}
	if newName == oldName {
		return
	}
	if projectStopped(out.Status) {
		return
	}

	nameAttr := check.nameAttr
	if nameAttr == "" {
		nameAttr = "name"
	}
	remedy := check.remedy
	if remedy == "" {
		remedy = renameRemedyLifecycle
	}
	resp.Diagnostics.AddAttributeError(
		path.Root(nameAttr),
		renameRequiresStopSummary,
		fmt.Sprintf("Renaming %q to %q would fail during apply: Arcane only renames a project that is stopped, and this one is %s.\n\n%s",
			oldName, newName, out.Status, remedy),
	)
}

// stopProjectForRename brings the project down unless it is already stopped,
// and reports whether it actually stopped anything. Callers use that to decide
// whether the project has to be started again once the rename went through.
func stopProjectForRename(ctx context.Context, client *sdkclient.Client, envID, projID string) (bool, error) {
	out, err := client.GetProject(ctx, envID, projID)
	if err != nil {
		return false, err
	}
	if projectStopped(out.Status) {
		return false, nil
	}
	if err := client.DownProject(ctx, envID, projID); err != nil {
		return false, err
	}
	return true, nil
}

// renameProject renames a project on behalf of a resource that does not manage
// the project lifecycle itself: it stops the project when the configuration
// allows it, renames it, and starts it again afterwards. A rename that would be
// rejected returns the same explanation the plan phase gives, because state
// that was never refreshed can carry a stale status into the apply.
//
// It is a no-op when the project already carries the wanted name, so callers
// can hand it the configured name on every update.
func renameProject(ctx context.Context, client *sdkclient.Client, envID, projID, newName string, stopBeforeRename bool) error {
	out, err := client.GetProject(ctx, envID, projID)
	if err != nil {
		return err
	}
	if out.Name == newName {
		return nil
	}

	restart := false
	if !projectStopped(out.Status) {
		if !stopBeforeRename {
			return fmt.Errorf("%s: renaming %q to %q needs a stopped project, and this one is %s.\n\n%s",
				renameRequiresStopSummary, out.Name, newName, out.Status, renameRemedyStopOnly)
		}
		if err := client.DownProject(ctx, envID, projID); err != nil {
			return fmt.Errorf("stopping the project before renaming it failed: %w", err)
		}
		restart = true
	}

	if _, err := client.UpdateProject(ctx, envID, projID, sdkclient.ProjectUpdateRequest{Name: &newName}); err != nil {
		if restart {
			// Say so: the project is down and the rename it was stopped for
			// never happened, so a retry is what brings it back up.
			return fmt.Errorf("renaming the project failed, and it is left stopped: %w", err)
		}
		return err
	}

	if restart {
		if err := client.UpProject(ctx, envID, projID, nil); err != nil {
			return fmt.Errorf("starting the project again after renaming it failed: %w", err)
		}
	}
	return nil
}
