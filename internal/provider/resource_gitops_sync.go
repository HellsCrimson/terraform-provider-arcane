package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &GitOpsSyncResource{}
var _ resource.ResourceWithImportState = &GitOpsSyncResource{}
var _ resource.ResourceWithModifyPlan = &GitOpsSyncResource{}

type GitOpsSyncResource struct {
	client *sdkclient.Client
}

func NewGitOpsSyncResource() resource.Resource {
	return &GitOpsSyncResource{}
}

func (r *GitOpsSyncResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gitops_sync"
}

func (r *GitOpsSyncResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "GitOps sync ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"environment_id": resourceschema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": resourceschema.StringAttribute{
				Required:    true,
				Description: "Sync configuration name",
			},
			"repository_id": resourceschema.StringAttribute{
				Required:    true,
				Description: "Git repository ID",
			},
			"branch": resourceschema.StringAttribute{
				Required:    true,
				Description: "Git branch to sync",
			},
			"compose_path": resourceschema.StringAttribute{
				Required:    true,
				Description: "Path to docker-compose file in the repository",
			},
			"project_name": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Project name for the synced compose stack",
			},
			"auto_sync": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Enable automatic sync on interval",
			},
			"sync_interval": resourceschema.Int64Attribute{
				Optional:    true,
				Description: "Sync interval in seconds",
			},
			"max_sync_binary_size": resourceschema.Int64Attribute{
				Optional:    true,
				Description: "Maximum binary file size to sync in bytes",
			},
			"max_sync_files": resourceschema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of files to sync",
			},
			"max_sync_total_size": resourceschema.Int64Attribute{
				Optional:    true,
				Description: "Maximum total sync size in bytes",
			},
			"sync_directory": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Whether to sync the full directory instead of only the compose file",
			},
			"target_type": resourceschema.StringAttribute{
				Optional:    true,
				Description: "GitOps sync target type",
			},
			"enabled": resourceschema.BoolAttribute{
				Computed:    true,
				Description: "Whether the sync is enabled (read-only)",
			},
			"environment_variables": resourceschema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Environment variables for the synced project",
			},
			"start_project": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Whether to start the project after creation (default: true). This is not sent to the API but controls lifecycle behavior.",
			},
			"fail_if_name_exists": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "If true, fail during the plan phase when a GitOps sync with the same name already exists in the environment, instead of creating a duplicate. Defaults to false.",
			},
			"stop_before_rename": resourceschema.BoolAttribute{
				Optional:    true,
				Description: stopBeforeRenameDescription + " Applies to a project_name change, which renames the project this sync is bound to.",
			},
			"pre_deploy_script_path": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Path inside the synced repository to a script executed in a throwaway container before each deploy",
			},
			"pre_deploy_runner_image": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Container image used to run the pre-deploy script. Required by the API whenever pre_deploy_script_path is set",
			},
			"pre_deploy_env": resourceschema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Environment variables exposed to the pre-deploy script, one KEY=VALUE entry per line (.env file format). Marked sensitive because it commonly carries key material (e.g. SOPS_AGE_KEY)",
			},
			"pre_deploy_extra_mounts": resourceschema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Extra bind mounts for the pre-deploy runner container, one entry per line in docker 'src:tgt[:ro|:rw]' form",
			},
			"pre_deploy_timeout_sec": resourceschema.Int64Attribute{
				Optional:    true,
				Description: "Timeout in seconds for the pre-deploy script (server default 60, capped by the server-side maximum)",
			},
			"pre_deploy_network_mode": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Docker network mode for the pre-deploy runner container: \"none\" (server default), \"bridge\", \"host\", or a Docker network name",
			},

			// Computed fields
			"project_id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Associated project ID",
			},
			"last_sync_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Last sync timestamp",
			},
			"last_sync_commit": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Last synced commit hash",
			},
			"last_sync_status": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Last sync status",
			},
			"last_sync_error": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Last sync error message",
			},
			"created_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// preDeployString resolves the post-apply value of an optional pre-deploy
// string attribute. The planned value wins, because Terraform requires the
// state of an Optional, non-Computed attribute to match the configuration
// exactly. The exception is a value that was not resolvable at plan time: an
// unknown left in state fails the apply outright, so the server's answer is
// the only value available there.
func preDeployString(planned types.String, server *string) types.String {
	if !planned.IsUnknown() {
		return planned
	}
	if server == nil {
		return types.StringNull()
	}
	return nullableString(*server)
}

func (r *GitOpsSyncResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

// ModifyPlan enforces the checks that belong in the plan phase: the optional
// fail_if_name_exists guard on create, and the stop-before-rename requirement
// a project_name change runs into.
func (r *GitOpsSyncResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil || req.Plan.Raw.IsNull() {
		return
	}
	if req.State.Raw.IsNull() {
		r.planFailIfNameExists(ctx, req, resp)
		return
	}
	r.planRenameRequiresStop(ctx, req, resp)
}

// planFailIfNameExists fails the plan when fail_if_name_exists is set and a
// GitOps sync with the same name already exists in the environment, instead of
// silently creating a duplicate.
func (r *GitOpsSyncResource) planFailIfNameExists(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var plan gitOpsSyncModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.FailIfNameExists.ValueBool() {
		return
	}
	if plan.Name.IsUnknown() || plan.Name.IsNull() || plan.EnvironmentID.IsUnknown() || plan.EnvironmentID.IsNull() {
		return
	}

	name := plan.Name.ValueString()
	envID := plan.EnvironmentID.ValueString()
	existing, err := r.client.ListGitOpsSyncs(ctx, envID)
	if err != nil {
		resp.Diagnostics.AddError("failed to check for existing gitops sync name", err.Error())
		return
	}
	for _, s := range existing {
		if s.Name == name {
			resp.Diagnostics.AddError(
				"gitops sync name already exists",
				fmt.Sprintf("A GitOps sync named %q already exists in environment %q (id %q). Choose a unique name, import the existing sync, or set fail_if_name_exists = false.", name, envID, s.ID),
			)
			return
		}
	}
}

// planRenameRequiresStop fails the plan when a project_name change renames a
// project that Arcane would refuse to rename because it is not stopped. The
// rename is the provider's own doing: Arcane stores project_name on the sync
// record and never renames the bound project itself, so Update does it (see
// renameProject in project_rename.go).
func (r *GitOpsSyncResource) planRenameRequiresStop(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var plan, state gitOpsSyncModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.ID.IsUnknown() {
		// The resource is being replaced, so the sync is recreated and its
		// project comes from a fresh sync rather than from a rename.
		return
	}

	planRenameRequiresStop(ctx, r.client, resp, renameCheck{
		envID: state.EnvironmentID.ValueString(),
		// Empty until the first sync created the project; there is nothing to
		// rename before that, and the project is created under the new name.
		projID: state.ProjectID.ValueString(),
		// state.project_name holds the configured value, which is null when the
		// user never set one: leave it empty so the API answers for the name.
		oldName:          state.ProjectName.ValueString(),
		newName:          plan.ProjectName,
		stopBeforeRename: boolValue(plan.StopBeforeRename),
		nameAttr:         "project_name",
		remedy:           renameRemedyStopOnly,
	})
}

type gitOpsSyncModel struct {
	ID                   types.String `tfsdk:"id"`
	EnvironmentID        types.String `tfsdk:"environment_id"`
	Name                 types.String `tfsdk:"name"`
	RepositoryID         types.String `tfsdk:"repository_id"`
	Branch               types.String `tfsdk:"branch"`
	ComposePath          types.String `tfsdk:"compose_path"`
	ProjectName          types.String `tfsdk:"project_name"`
	AutoSync             types.Bool   `tfsdk:"auto_sync"`
	SyncInterval         types.Int64  `tfsdk:"sync_interval"`
	MaxSyncBinarySize    types.Int64  `tfsdk:"max_sync_binary_size"`
	MaxSyncFiles         types.Int64  `tfsdk:"max_sync_files"`
	MaxSyncTotalSize     types.Int64  `tfsdk:"max_sync_total_size"`
	SyncDirectory        types.Bool   `tfsdk:"sync_directory"`
	TargetType           types.String `tfsdk:"target_type"`
	PreDeployScriptPath  types.String `tfsdk:"pre_deploy_script_path"`
	PreDeployRunnerImage types.String `tfsdk:"pre_deploy_runner_image"`
	PreDeployEnv         types.String `tfsdk:"pre_deploy_env"`
	PreDeployExtraMounts types.String `tfsdk:"pre_deploy_extra_mounts"`
	PreDeployTimeoutSec  types.Int64  `tfsdk:"pre_deploy_timeout_sec"`
	PreDeployNetworkMode types.String `tfsdk:"pre_deploy_network_mode"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	EnvironmentVariables types.Map    `tfsdk:"environment_variables"`
	StartProject         types.Bool   `tfsdk:"start_project"`
	FailIfNameExists     types.Bool   `tfsdk:"fail_if_name_exists"`
	StopBeforeRename     types.Bool   `tfsdk:"stop_before_rename"`
	ProjectID            types.String `tfsdk:"project_id"`
	LastSyncAt           types.String `tfsdk:"last_sync_at"`
	LastSyncCommit       types.String `tfsdk:"last_sync_commit"`
	LastSyncStatus       types.String `tfsdk:"last_sync_status"`
	LastSyncError        types.String `tfsdk:"last_sync_error"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

// mapToEnvContent converts a Terraform map to .env file format
func mapToEnvContent(ctx context.Context, envMap types.Map) (string, error) {
	if envMap.IsNull() || envMap.IsUnknown() {
		return "", nil
	}

	elements := make(map[string]types.String, len(envMap.Elements()))
	diags := envMap.ElementsAs(ctx, &elements, false)
	if diags.HasError() {
		return "", fmt.Errorf("failed to convert map elements")
	}

	var lines []string
	for key, value := range elements {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value.ValueString()))
	}
	return strings.Join(lines, "\n"), nil
}

// envContentToMap converts .env file format to a Terraform map
func envContentToMap(ctx context.Context, envContent string) (types.Map, error) {
	if envContent == "" {
		return types.MapNull(types.StringType), nil
	}

	envVars := make(map[string]string)
	lines := strings.Split(envContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		}
	}

	if len(envVars) == 0 {
		return types.MapNull(types.StringType), nil
	}

	result, diags := types.MapValueFrom(ctx, types.StringType, envVars)
	if diags.HasError() {
		return types.MapNull(types.StringType), fmt.Errorf("failed to create map from environment variables")
	}
	return result, nil
}

func (r *GitOpsSyncResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gitOpsSyncModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.GitOpsSyncCreateRequest{
		Name:         plan.Name.ValueString(),
		RepositoryID: plan.RepositoryID.ValueString(),
		Branch:       plan.Branch.ValueString(),
		ComposePath:  plan.ComposePath.ValueString(),
	}

	if !plan.ProjectName.IsNull() && !plan.ProjectName.IsUnknown() {
		v := plan.ProjectName.ValueString()
		body.ProjectName = &v
	}
	if !plan.AutoSync.IsNull() && !plan.AutoSync.IsUnknown() {
		v := plan.AutoSync.ValueBool()
		body.AutoSync = &v
	}
	if !plan.SyncInterval.IsNull() && !plan.SyncInterval.IsUnknown() {
		v := plan.SyncInterval.ValueInt64()
		body.SyncInterval = &v
	}
	if !plan.MaxSyncBinarySize.IsNull() && !plan.MaxSyncBinarySize.IsUnknown() {
		v := plan.MaxSyncBinarySize.ValueInt64()
		body.MaxSyncBinarySize = &v
	}
	if !plan.MaxSyncFiles.IsNull() && !plan.MaxSyncFiles.IsUnknown() {
		v := plan.MaxSyncFiles.ValueInt64()
		body.MaxSyncFiles = &v
	}
	if !plan.MaxSyncTotalSize.IsNull() && !plan.MaxSyncTotalSize.IsUnknown() {
		v := plan.MaxSyncTotalSize.ValueInt64()
		body.MaxSyncTotalSize = &v
	}
	if !plan.SyncDirectory.IsNull() && !plan.SyncDirectory.IsUnknown() {
		v := plan.SyncDirectory.ValueBool()
		body.SyncDirectory = &v
	}
	if !plan.TargetType.IsNull() && !plan.TargetType.IsUnknown() {
		v := plan.TargetType.ValueString()
		body.TargetType = &v
	}
	if !plan.PreDeployScriptPath.IsNull() && !plan.PreDeployScriptPath.IsUnknown() {
		v := plan.PreDeployScriptPath.ValueString()
		body.PreDeployScriptPath = &v
	}
	if !plan.PreDeployRunnerImage.IsNull() && !plan.PreDeployRunnerImage.IsUnknown() {
		v := plan.PreDeployRunnerImage.ValueString()
		body.PreDeployRunnerImage = &v
	}
	if !plan.PreDeployEnv.IsNull() && !plan.PreDeployEnv.IsUnknown() {
		v := plan.PreDeployEnv.ValueString()
		body.PreDeployEnv = &v
	}
	if !plan.PreDeployExtraMounts.IsNull() && !plan.PreDeployExtraMounts.IsUnknown() {
		v := plan.PreDeployExtraMounts.ValueString()
		body.PreDeployExtraMounts = &v
	}
	if !plan.PreDeployTimeoutSec.IsNull() && !plan.PreDeployTimeoutSec.IsUnknown() {
		v := plan.PreDeployTimeoutSec.ValueInt64()
		body.PreDeployTimeoutSec = &v
	}
	if !plan.PreDeployNetworkMode.IsNull() && !plan.PreDeployNetworkMode.IsUnknown() {
		v := plan.PreDeployNetworkMode.ValueString()
		body.PreDeployNetworkMode = &v
	}
	// Note: 'enabled' is read-only and not part of the create request

	sync, err := r.client.CreateGitOpsSync(ctx, plan.EnvironmentID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("create gitops sync failed", err.Error())
		return
	}

	// If environment variables are provided and a project was created, update the project with env content
	if !plan.EnvironmentVariables.IsNull() && !plan.EnvironmentVariables.IsUnknown() && sync.ProjectID != nil {
		envContent, err := mapToEnvContent(ctx, plan.EnvironmentVariables)
		if err != nil {
			resp.Diagnostics.AddError("convert environment variables to .env format failed", err.Error())
			return
		}
		projectUpdateBody := sdkclient.ProjectUpdateRequest{
			EnvContent: &envContent,
		}
		_, err = r.client.UpdateProject(ctx, plan.EnvironmentID.ValueString(), *sync.ProjectID, projectUpdateBody)
		if err != nil {
			resp.Diagnostics.AddError("update project env content failed", err.Error())
			return
		}
	}

	// Start the project if requested (default: true)
	startProject := true // default value
	if !plan.StartProject.IsNull() && !plan.StartProject.IsUnknown() {
		startProject = plan.StartProject.ValueBool()
	}
	if startProject && sync.ProjectID != nil {
		err = r.client.DeployProject(ctx, plan.EnvironmentID.ValueString(), *sync.ProjectID)
		if err != nil {
			resp.Diagnostics.AddError("deploy project failed", err.Error())
			return
		}
	}

	state := gitOpsSyncModel{
		ID:            types.StringValue(sync.ID),
		EnvironmentID: types.StringValue(sync.EnvironmentID),
		Name:          types.StringValue(sync.Name),
		RepositoryID:  types.StringValue(sync.RepositoryID),
		Branch:        types.StringValue(sync.Branch),
		ComposePath:   types.StringValue(sync.ComposePath),
		// Optional fields below are set conditionally so we preserve
		// the user's plan (null) when they didn't provide a value.
		EnvironmentVariables: plan.EnvironmentVariables,
		StartProject:         plan.StartProject,     // Preserve the user's preference
		FailIfNameExists:     plan.FailIfNameExists, // Plan-time-only guard, preserve as configured
		StopBeforeRename:     plan.StopBeforeRename, // Rename behavior, preserve as configured
		CreatedAt:            types.StringValue(sync.CreatedAt),
		UpdatedAt:            types.StringValue(sync.UpdatedAt),
	}

	// Preserve plan nulls for optional fields: use server value only when
	// the user provided a value in the plan; otherwise keep plan (null).
	if !plan.ProjectName.IsNull() && !plan.ProjectName.IsUnknown() {
		state.ProjectName = types.StringValue(sync.ProjectName)
	} else {
		state.ProjectName = plan.ProjectName
	}
	if !plan.AutoSync.IsNull() && !plan.AutoSync.IsUnknown() {
		state.AutoSync = types.BoolValue(sync.AutoSync)
	} else {
		state.AutoSync = plan.AutoSync
	}
	if !plan.SyncInterval.IsNull() && !plan.SyncInterval.IsUnknown() {
		state.SyncInterval = types.Int64Value(sync.SyncInterval)
	} else {
		state.SyncInterval = plan.SyncInterval
	}
	if !plan.MaxSyncBinarySize.IsNull() && !plan.MaxSyncBinarySize.IsUnknown() {
		state.MaxSyncBinarySize = types.Int64Value(sync.MaxSyncBinarySize)
	} else {
		state.MaxSyncBinarySize = plan.MaxSyncBinarySize
	}
	if !plan.MaxSyncFiles.IsNull() && !plan.MaxSyncFiles.IsUnknown() {
		state.MaxSyncFiles = types.Int64Value(sync.MaxSyncFiles)
	} else {
		state.MaxSyncFiles = plan.MaxSyncFiles
	}
	if !plan.MaxSyncTotalSize.IsNull() && !plan.MaxSyncTotalSize.IsUnknown() {
		state.MaxSyncTotalSize = types.Int64Value(sync.MaxSyncTotalSize)
	} else {
		state.MaxSyncTotalSize = plan.MaxSyncTotalSize
	}
	if !plan.SyncDirectory.IsNull() && !plan.SyncDirectory.IsUnknown() {
		state.SyncDirectory = types.BoolValue(sync.SyncDirectory)
	} else {
		state.SyncDirectory = plan.SyncDirectory
	}
	if !plan.TargetType.IsNull() && !plan.TargetType.IsUnknown() {
		state.TargetType = types.StringValue(sync.TargetType)
	} else {
		state.TargetType = plan.TargetType
	}
	// pre_deploy_* are Optional and not Computed, so the post-apply state has to
	// equal the configuration exactly. Keep the planned value rather than the
	// server's answer: that answer carries defaults (60, "none") for attributes
	// nobody configured, and may clamp or normalise the ones that were set.
	// Read surfaces any server-side divergence as ordinary drift instead.
	state.PreDeployScriptPath = preDeployString(plan.PreDeployScriptPath, sync.PreDeployScriptPath)
	state.PreDeployRunnerImage = preDeployString(plan.PreDeployRunnerImage, sync.PreDeployRunnerImage)
	state.PreDeployEnv = preDeployString(plan.PreDeployEnv, sync.PreDeployEnv)
	state.PreDeployExtraMounts = preDeployString(plan.PreDeployExtraMounts, sync.PreDeployExtraMounts)
	if plan.PreDeployTimeoutSec.IsUnknown() {
		state.PreDeployTimeoutSec = types.Int64Value(sync.PreDeployTimeoutSec)
	} else {
		state.PreDeployTimeoutSec = plan.PreDeployTimeoutSec
	}
	state.PreDeployNetworkMode = preDeployString(plan.PreDeployNetworkMode, &sync.PreDeployNetworkMode)

	if sync.ProjectID != nil {
		state.ProjectID = types.StringValue(*sync.ProjectID)
	}
	if sync.LastSyncAt != nil {
		state.LastSyncAt = types.StringValue(*sync.LastSyncAt)
	}
	if sync.LastSyncCommit != nil {
		state.LastSyncCommit = types.StringValue(*sync.LastSyncCommit)
	}
	if sync.LastSyncStatus != nil {
		state.LastSyncStatus = types.StringValue(*sync.LastSyncStatus)
	}
	if sync.LastSyncError != nil {
		state.LastSyncError = types.StringValue(*sync.LastSyncError)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GitOpsSyncResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gitOpsSyncModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sync, err := r.client.GetGitOpsSync(ctx, state.EnvironmentID.ValueString(), state.ID.ValueString())
	if err != nil {
		if r.client.IsResourceGone(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read gitops sync failed", err.Error())
		return
	}

	state.Name = types.StringValue(sync.Name)
	state.RepositoryID = types.StringValue(sync.RepositoryID)
	state.Branch = types.StringValue(sync.Branch)
	state.ComposePath = types.StringValue(sync.ComposePath)
	// Preserve existing state nulls for optional fields: only overwrite
	// with server values when the state already had a non-null value.
	if !state.ProjectName.IsNull() && !state.ProjectName.IsUnknown() {
		state.ProjectName = types.StringValue(sync.ProjectName)
	}
	if !state.AutoSync.IsNull() && !state.AutoSync.IsUnknown() {
		state.AutoSync = types.BoolValue(sync.AutoSync)
	}
	if !state.SyncInterval.IsNull() && !state.SyncInterval.IsUnknown() {
		state.SyncInterval = types.Int64Value(sync.SyncInterval)
	}
	if !state.MaxSyncBinarySize.IsNull() && !state.MaxSyncBinarySize.IsUnknown() {
		state.MaxSyncBinarySize = types.Int64Value(sync.MaxSyncBinarySize)
	}
	if !state.MaxSyncFiles.IsNull() && !state.MaxSyncFiles.IsUnknown() {
		state.MaxSyncFiles = types.Int64Value(sync.MaxSyncFiles)
	}
	if !state.MaxSyncTotalSize.IsNull() && !state.MaxSyncTotalSize.IsUnknown() {
		state.MaxSyncTotalSize = types.Int64Value(sync.MaxSyncTotalSize)
	}
	if !state.SyncDirectory.IsNull() && !state.SyncDirectory.IsUnknown() {
		state.SyncDirectory = types.BoolValue(sync.SyncDirectory)
	}
	if !state.TargetType.IsNull() && !state.TargetType.IsUnknown() {
		state.TargetType = types.StringValue(sync.TargetType)
	}
	if !state.PreDeployScriptPath.IsNull() && !state.PreDeployScriptPath.IsUnknown() {
		state.PreDeployScriptPath = types.StringPointerValue(sync.PreDeployScriptPath)
	}
	if !state.PreDeployRunnerImage.IsNull() && !state.PreDeployRunnerImage.IsUnknown() {
		state.PreDeployRunnerImage = types.StringPointerValue(sync.PreDeployRunnerImage)
	}
	if !state.PreDeployEnv.IsNull() && !state.PreDeployEnv.IsUnknown() {
		state.PreDeployEnv = types.StringPointerValue(sync.PreDeployEnv)
	}
	if !state.PreDeployExtraMounts.IsNull() && !state.PreDeployExtraMounts.IsUnknown() {
		state.PreDeployExtraMounts = types.StringPointerValue(sync.PreDeployExtraMounts)
	}
	// The API always reports server defaults for these two; only reflect them
	// when the practitioner configured a value, so null stays null.
	if !state.PreDeployTimeoutSec.IsNull() && !state.PreDeployTimeoutSec.IsUnknown() {
		state.PreDeployTimeoutSec = types.Int64Value(sync.PreDeployTimeoutSec)
	}
	if !state.PreDeployNetworkMode.IsNull() && !state.PreDeployNetworkMode.IsUnknown() {
		state.PreDeployNetworkMode = types.StringValue(sync.PreDeployNetworkMode)
	}
	state.Enabled = types.BoolValue(sync.Enabled)
	// Leave updated_at and created_at unchanged to avoid plan inconsistency on server-side timestamp changes
	// start_project is preserved from state as it's a lifecycle control, not an API field

	if sync.ProjectID != nil {
		state.ProjectID = types.StringValue(*sync.ProjectID)
	} else {
		state.ProjectID = types.StringNull()
	}
	// Preserve the configured environment variables in state so refreshes do
	// not turn them into a fresh API-derived value every time.
	if sync.LastSyncAt != nil {
		state.LastSyncAt = types.StringValue(*sync.LastSyncAt)
	} else {
		state.LastSyncAt = types.StringNull()
	}
	if sync.LastSyncCommit != nil {
		state.LastSyncCommit = types.StringValue(*sync.LastSyncCommit)
	} else {
		state.LastSyncCommit = types.StringNull()
	}
	if sync.LastSyncStatus != nil {
		state.LastSyncStatus = types.StringValue(*sync.LastSyncStatus)
	} else {
		state.LastSyncStatus = types.StringNull()
	}
	if sync.LastSyncError != nil && *sync.LastSyncError != "" {
		state.LastSyncError = types.StringValue(*sync.LastSyncError)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GitOpsSyncResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan gitOpsSyncModel
	var state gitOpsSyncModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.GitOpsSyncUpdateRequest{}

	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		v := plan.Name.ValueString()
		body.Name = &v
	}
	if !plan.RepositoryID.IsNull() && !plan.RepositoryID.IsUnknown() {
		v := plan.RepositoryID.ValueString()
		body.RepositoryID = &v
	}
	if !plan.Branch.IsNull() && !plan.Branch.IsUnknown() {
		v := plan.Branch.ValueString()
		body.Branch = &v
	}
	if !plan.ComposePath.IsNull() && !plan.ComposePath.IsUnknown() {
		v := plan.ComposePath.ValueString()
		body.ComposePath = &v
	}
	if !plan.ProjectName.IsNull() && !plan.ProjectName.IsUnknown() {
		v := plan.ProjectName.ValueString()
		body.ProjectName = &v
	}
	if !plan.AutoSync.IsNull() && !plan.AutoSync.IsUnknown() {
		v := plan.AutoSync.ValueBool()
		body.AutoSync = &v
	}
	if !plan.SyncInterval.IsNull() && !plan.SyncInterval.IsUnknown() {
		v := plan.SyncInterval.ValueInt64()
		body.SyncInterval = &v
	}
	if !plan.MaxSyncBinarySize.IsNull() && !plan.MaxSyncBinarySize.IsUnknown() {
		v := plan.MaxSyncBinarySize.ValueInt64()
		body.MaxSyncBinarySize = &v
	}
	if !plan.MaxSyncFiles.IsNull() && !plan.MaxSyncFiles.IsUnknown() {
		v := plan.MaxSyncFiles.ValueInt64()
		body.MaxSyncFiles = &v
	}
	if !plan.MaxSyncTotalSize.IsNull() && !plan.MaxSyncTotalSize.IsUnknown() {
		v := plan.MaxSyncTotalSize.ValueInt64()
		body.MaxSyncTotalSize = &v
	}
	if !plan.SyncDirectory.IsNull() && !plan.SyncDirectory.IsUnknown() {
		v := plan.SyncDirectory.ValueBool()
		body.SyncDirectory = &v
	}
	if !plan.TargetType.IsNull() && !plan.TargetType.IsUnknown() {
		v := plan.TargetType.ValueString()
		body.TargetType = &v
	}
	if !plan.PreDeployScriptPath.IsNull() && !plan.PreDeployScriptPath.IsUnknown() {
		v := plan.PreDeployScriptPath.ValueString()
		body.PreDeployScriptPath = &v
	} else if plan.PreDeployScriptPath.IsNull() && !state.PreDeployScriptPath.IsNull() {
		// The attribute was removed from the configuration: an empty string
		// clears the hook on the server (omitting the field would keep it).
		v := ""
		body.PreDeployScriptPath = &v
	}
	if !plan.PreDeployRunnerImage.IsNull() && !plan.PreDeployRunnerImage.IsUnknown() {
		v := plan.PreDeployRunnerImage.ValueString()
		body.PreDeployRunnerImage = &v
	} else if plan.PreDeployRunnerImage.IsNull() && !state.PreDeployRunnerImage.IsNull() {
		v := ""
		body.PreDeployRunnerImage = &v
	}
	if !plan.PreDeployEnv.IsNull() && !plan.PreDeployEnv.IsUnknown() {
		v := plan.PreDeployEnv.ValueString()
		body.PreDeployEnv = &v
	} else if plan.PreDeployEnv.IsNull() && !state.PreDeployEnv.IsNull() {
		v := ""
		body.PreDeployEnv = &v
	}
	if !plan.PreDeployExtraMounts.IsNull() && !plan.PreDeployExtraMounts.IsUnknown() {
		v := plan.PreDeployExtraMounts.ValueString()
		body.PreDeployExtraMounts = &v
	} else if plan.PreDeployExtraMounts.IsNull() && !state.PreDeployExtraMounts.IsNull() {
		v := ""
		body.PreDeployExtraMounts = &v
	}
	if !plan.PreDeployTimeoutSec.IsNull() && !plan.PreDeployTimeoutSec.IsUnknown() {
		v := plan.PreDeployTimeoutSec.ValueInt64()
		body.PreDeployTimeoutSec = &v
	} else if plan.PreDeployTimeoutSec.IsNull() && !state.PreDeployTimeoutSec.IsNull() {
		// Zero resets the server to its default (60), the same way an empty
		// string resets the pre-deploy strings. The field is a *int64, so a
		// pointer to zero is still encoded rather than dropped by omitempty.
		v := int64(0)
		body.PreDeployTimeoutSec = &v
	}
	if !plan.PreDeployNetworkMode.IsNull() && !plan.PreDeployNetworkMode.IsUnknown() {
		v := plan.PreDeployNetworkMode.ValueString()
		body.PreDeployNetworkMode = &v
	} else if plan.PreDeployNetworkMode.IsNull() && !state.PreDeployNetworkMode.IsNull() {
		// Empty string resets the server to its default ("none").
		v := ""
		body.PreDeployNetworkMode = &v
	}
	// Note: 'enabled' is read-only and not part of the update request

	sync, err := r.client.UpdateGitOpsSync(ctx, state.EnvironmentID.ValueString(), state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("update gitops sync failed", err.Error())
		return
	}

	// Arcane never renames the project a sync is bound to: UpdateSync only
	// writes project_name onto the sync record, and a sync run looks the project
	// up by ID. So a project_name change only reaches the project because the
	// provider renames it here, under the stopped-project rule Arcane enforces
	// (see project_rename.go; ModifyPlan rejects renames this cannot satisfy).
	//
	// Only a changed project_name renames: that keeps the apply doing exactly
	// what the plan checked, and leaves a project whose name drifted from the
	// sync record alone until the practitioner asks for it.
	if sync.ProjectID != nil && !plan.ProjectName.IsNull() && !plan.ProjectName.IsUnknown() && !plan.ProjectName.Equal(state.ProjectName) {
		if err := renameProject(ctx, r.client, state.EnvironmentID.ValueString(), *sync.ProjectID, plan.ProjectName.ValueString(), boolValue(plan.StopBeforeRename)); err != nil {
			resp.Diagnostics.AddError("rename gitops sync project failed", err.Error())
			return
		}
	}

	// If environment variables changed and a project exists, update the project with env content
	if sync.ProjectID != nil && !plan.EnvironmentVariables.Equal(state.EnvironmentVariables) {
		projectUpdateBody := sdkclient.ProjectUpdateRequest{}
		if !plan.EnvironmentVariables.IsNull() && !plan.EnvironmentVariables.IsUnknown() {
			envContent, err := mapToEnvContent(ctx, plan.EnvironmentVariables)
			if err != nil {
				resp.Diagnostics.AddError("convert environment variables to .env format failed", err.Error())
				return
			}
			projectUpdateBody.EnvContent = &envContent
		} else {
			// Set to empty string if environment variables are being removed
			emptyContent := ""
			projectUpdateBody.EnvContent = &emptyContent
		}
		_, err := r.client.UpdateProject(ctx, state.EnvironmentID.ValueString(), *sync.ProjectID, projectUpdateBody)
		if err != nil {
			resp.Diagnostics.AddError("update project env content failed", err.Error())
			return
		}
	}

	state.Name = types.StringValue(sync.Name)
	state.RepositoryID = types.StringValue(sync.RepositoryID)
	state.Branch = types.StringValue(sync.Branch)
	state.ComposePath = types.StringValue(sync.ComposePath)
	// Preserve plan nulls for optional fields
	if !plan.ProjectName.IsNull() && !plan.ProjectName.IsUnknown() {
		state.ProjectName = types.StringValue(sync.ProjectName)
	} else {
		state.ProjectName = plan.ProjectName
	}
	if !plan.AutoSync.IsNull() && !plan.AutoSync.IsUnknown() {
		state.AutoSync = types.BoolValue(sync.AutoSync)
	} else {
		state.AutoSync = plan.AutoSync
	}
	if !plan.SyncInterval.IsNull() && !plan.SyncInterval.IsUnknown() {
		state.SyncInterval = types.Int64Value(sync.SyncInterval)
	} else {
		state.SyncInterval = plan.SyncInterval
	}
	if !plan.MaxSyncBinarySize.IsNull() && !plan.MaxSyncBinarySize.IsUnknown() {
		state.MaxSyncBinarySize = types.Int64Value(sync.MaxSyncBinarySize)
	} else {
		state.MaxSyncBinarySize = plan.MaxSyncBinarySize
	}
	if !plan.MaxSyncFiles.IsNull() && !plan.MaxSyncFiles.IsUnknown() {
		state.MaxSyncFiles = types.Int64Value(sync.MaxSyncFiles)
	} else {
		state.MaxSyncFiles = plan.MaxSyncFiles
	}
	if !plan.MaxSyncTotalSize.IsNull() && !plan.MaxSyncTotalSize.IsUnknown() {
		state.MaxSyncTotalSize = types.Int64Value(sync.MaxSyncTotalSize)
	} else {
		state.MaxSyncTotalSize = plan.MaxSyncTotalSize
	}
	if !plan.SyncDirectory.IsNull() && !plan.SyncDirectory.IsUnknown() {
		state.SyncDirectory = types.BoolValue(sync.SyncDirectory)
	} else {
		state.SyncDirectory = plan.SyncDirectory
	}
	if !plan.TargetType.IsNull() && !plan.TargetType.IsUnknown() {
		state.TargetType = types.StringValue(sync.TargetType)
	} else {
		state.TargetType = plan.TargetType
	}
	// pre_deploy_* are Optional and not Computed, so the post-apply state has to
	// equal the configuration exactly. Keep the planned value rather than the
	// server's answer: that answer carries defaults (60, "none") for attributes
	// nobody configured, and may clamp or normalise the ones that were set.
	// Read surfaces any server-side divergence as ordinary drift instead.
	state.PreDeployScriptPath = preDeployString(plan.PreDeployScriptPath, sync.PreDeployScriptPath)
	state.PreDeployRunnerImage = preDeployString(plan.PreDeployRunnerImage, sync.PreDeployRunnerImage)
	state.PreDeployEnv = preDeployString(plan.PreDeployEnv, sync.PreDeployEnv)
	state.PreDeployExtraMounts = preDeployString(plan.PreDeployExtraMounts, sync.PreDeployExtraMounts)
	if plan.PreDeployTimeoutSec.IsUnknown() {
		state.PreDeployTimeoutSec = types.Int64Value(sync.PreDeployTimeoutSec)
	} else {
		state.PreDeployTimeoutSec = plan.PreDeployTimeoutSec
	}
	state.PreDeployNetworkMode = preDeployString(plan.PreDeployNetworkMode, &sync.PreDeployNetworkMode)
	state.Enabled = types.BoolValue(sync.Enabled)
	// Leave updated_at unchanged to avoid plan inconsistency on server-side timestamp changes
	state.EnvironmentVariables = plan.EnvironmentVariables
	state.StartProject = plan.StartProject         // Preserve the user's preference
	state.FailIfNameExists = plan.FailIfNameExists // Plan-time-only guard, preserve as configured
	state.StopBeforeRename = plan.StopBeforeRename // Rename behavior, preserve as configured

	if sync.ProjectID != nil {
		state.ProjectID = types.StringValue(*sync.ProjectID)
	} else {
		state.ProjectID = types.StringNull()
		state.EnvironmentVariables = types.MapNull(types.StringType)
	}
	if sync.LastSyncAt != nil {
		state.LastSyncAt = types.StringValue(*sync.LastSyncAt)
	} else {
		state.LastSyncAt = types.StringNull()
	}
	if sync.LastSyncCommit != nil {
		state.LastSyncCommit = types.StringValue(*sync.LastSyncCommit)
	} else {
		state.LastSyncCommit = types.StringNull()
	}
	if sync.LastSyncStatus != nil {
		state.LastSyncStatus = types.StringValue(*sync.LastSyncStatus)
	} else {
		state.LastSyncStatus = types.StringNull()
	}
	if sync.LastSyncError != nil {
		state.LastSyncError = types.StringValue(*sync.LastSyncError)
	} else {
		state.LastSyncError = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GitOpsSyncResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gitOpsSyncModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envID := state.EnvironmentID.ValueString()
	syncID := state.ID.ValueString()

	// Prefer project_id from state, fallback to live lookup when state does not have it yet.
	projectID := ""
	if !state.ProjectID.IsNull() && !state.ProjectID.IsUnknown() {
		projectID = state.ProjectID.ValueString()
	}
	if projectID == "" {
		sync, err := r.client.GetGitOpsSync(ctx, envID, syncID)
		if err != nil {
			if !r.client.IsResourceGone(err) {
				resp.Diagnostics.AddError("read gitops sync before delete failed", err.Error())
			}
		} else if sync.ProjectID != nil {
			projectID = *sync.ProjectID
		}
	}

	if err := r.client.DeleteGitOpsSync(ctx, envID, syncID); err != nil {
		if r.client.IsResourceGone(err) {
			// Continue so we can still try to cleanup the project if we have an ID.
		} else {
			resp.Diagnostics.AddError("delete gitops sync failed", err.Error())
		}
	}

	if projectID != "" {
		opts := sdkclient.ProjectDestroyOptions{
			RemoveFiles:   false,
			RemoveVolumes: false,
		}
		if err := r.client.DestroyProject(ctx, envID, projectID, opts); err != nil {
			if r.client.IsResourceGone(err) {
				return
			}
			resp.Diagnostics.AddError("destroy gitops sync project failed", err.Error())
		}
	}
}

func (r *GitOpsSyncResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// envID:syncID
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("invalid import id", "expected env_id:sync_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
