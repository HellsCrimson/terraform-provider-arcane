package provider

import (
	"context"
	"sort"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	gitOpsModeDeploy        = "deploy"
	gitOpsModeBackup        = "backup"
	gitOpsModeBidirectional = "bidirectional"
)

var gitOpsModes = []string{gitOpsModeDeploy, gitOpsModeBackup, gitOpsModeBidirectional}

// gitOpsBackupMode reports whether mode pushes an existing project to Git
// (backup or bidirectional) instead of deploying a project from Git.
func gitOpsBackupMode(mode types.String) bool {
	v := mode.ValueString()
	return v == gitOpsModeBackup || v == gitOpsModeBidirectional
}

// ValidateConfig checks the attributes each mode needs: a deploy sync reads
// compose_path from the repository and creates its own project, a backup sync
// pushes an existing project (project_id) to backup_directory, where Arcane
// derives the compose path itself.
func (r *GitOpsSyncResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg gitOpsSyncModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() || cfg.Mode.IsUnknown() {
		return
	}
	if gitOpsBackupMode(cfg.Mode) {
		if cfg.ProjectID.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("project_id"), "missing project_id",
				"mode = \""+cfg.Mode.ValueString()+"\" backs up an existing project: set project_id.")
		}
		if cfg.BackupDirectory.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("backup_directory"), "missing backup_directory",
				"mode = \""+cfg.Mode.ValueString()+"\" needs the repository directory the project is pushed to.")
		}
		if !cfg.ComposePath.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("compose_path"), "compose_path not allowed",
				"In backup modes Arcane stores the compose file in backup_directory; remove compose_path.")
		}
		return
	}
	if cfg.ComposePath.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("compose_path"), "missing compose_path",
			"A deploy sync needs the path of the compose file in the repository.")
	}
	if !cfg.ProjectID.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("project_id"), "project_id not allowed",
			"A deploy sync creates its own project; project_id can only be set with mode = \"backup\" or \"bidirectional\".")
	}
}

// gitOpsBackupPaths converts the backup_paths set for a request. ok is false
// when the value is unknown and must not be sent.
func gitOpsBackupPaths(ctx context.Context, v types.Set) (*[]string, bool, diag.Diagnostics) {
	if v.IsUnknown() {
		return nil, false, nil
	}
	out := []string{}
	if v.IsNull() {
		return &out, true, nil
	}
	diags := v.ElementsAs(ctx, &out, false)
	return &out, true, diags
}

// gitOpsBackupInputs resolves the optional backup inputs for state: a
// configured value is echoed from the API, an unset one keeps desired's null.
func gitOpsBackupInputs(sync *sdkclient.GitOpsSync, desired gitOpsSyncModel) (dir, overridePath types.String, onSave types.Bool, paths types.Set) {
	dir = echoOptionalString(desired.BackupDirectory, sync.BackupDirectory)
	overridePath = echoOptionalString(desired.ComposeOverridePath, sync.ComposeOverridePath)
	onSave = desired.BackupOnSave
	if !onSave.IsNull() {
		onSave = types.BoolValue(sync.BackupOnSave)
	}
	paths = desired.BackupPaths
	if !paths.IsNull() {
		sorted := append([]string(nil), sync.BackupPaths...)
		sort.Strings(sorted)
		elems := make([]attr.Value, 0, len(sorted))
		for _, p := range sorted {
			elems = append(elems, types.StringValue(p))
		}
		paths = types.SetValueMust(types.StringType, elems)
	}
	return dir, overridePath, onSave, paths
}

// setGitOpsBackupFields copies the server-owned mode and backup status into
// state.
func setGitOpsBackupFields(state *gitOpsSyncModel, sync *sdkclient.GitOpsSync) {
	state.Mode = types.StringValue(sync.Mode)
	if state.Mode.ValueString() == "" {
		state.Mode = types.StringValue(gitOpsModeDeploy)
	}
	state.BackupPending = types.BoolValue(sync.BackupPending)
	state.BackupState = nullableString(sync.BackupState)
	state.BackupFailureReason = nullableString(sync.BackupFailureReason)
	state.LastBackupAt = types.StringPointerValue(sync.LastBackupAt)
}
