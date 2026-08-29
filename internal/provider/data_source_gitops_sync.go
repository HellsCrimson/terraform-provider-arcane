package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &GitOpsSyncDataSource{}

type GitOpsSyncDataSource struct {
	client *sdkclient.Client
}

func NewGitOpsSyncDataSource() datasource.DataSource {
	return &GitOpsSyncDataSource{}
}

func (d *GitOpsSyncDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gitops_sync"
}

func (d *GitOpsSyncDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane GitOps sync configuration",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "GitOps sync ID",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Sync configuration name",
			},
			"repository_id": schema.StringAttribute{
				Computed:    true,
				Description: "Git repository ID",
			},
			"branch": schema.StringAttribute{
				Computed:    true,
				Description: "Git branch",
			},
			"compose_path": schema.StringAttribute{
				Computed:    true,
				Description: "Path to docker-compose file",
			},
			"project_name": schema.StringAttribute{
				Computed:    true,
				Description: "Project name",
			},
			"auto_sync": schema.BoolAttribute{
				Computed:    true,
				Description: "Auto sync enabled",
			},
			"sync_interval": schema.Int64Attribute{
				Computed:    true,
				Description: "Sync interval in seconds",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether sync is enabled",
			},
			"pre_deploy_script_path": schema.StringAttribute{
				Computed:    true,
				Description: "Path inside the synced repository to the pre-deploy hook script",
			},
			"pre_deploy_runner_image": schema.StringAttribute{
				Computed:    true,
				Description: "Container image used to run the pre-deploy script",
			},
			"pre_deploy_env": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Environment variables exposed to the pre-deploy script (.env file format)",
			},
			"pre_deploy_extra_mounts": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Extra bind mounts for the pre-deploy runner container, one per line",
			},
			"pre_deploy_timeout_sec": schema.Int64Attribute{
				Computed:    true,
				Description: "Timeout in seconds for the pre-deploy script",
			},
			"pre_deploy_network_mode": schema.StringAttribute{
				Computed:    true,
				Description: "Docker network mode for the pre-deploy runner container",
			},
			"environment_variables": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Environment variables",
			},
			"project_id": schema.StringAttribute{
				Computed:    true,
				Description: "Associated project ID",
			},
			"last_sync_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last sync timestamp",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
			},
		},
	}
}

func (d *GitOpsSyncDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*sdkclient.Client)
	if !ok {
		resp.Diagnostics.AddError("unexpected provider data type", "Expected *sdkclient.Client")
		return
	}
	d.client = client
}

type gitOpsSyncDataSourceModel struct {
	EnvironmentID        types.String `tfsdk:"environment_id"`
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	RepositoryID         types.String `tfsdk:"repository_id"`
	Branch               types.String `tfsdk:"branch"`
	ComposePath          types.String `tfsdk:"compose_path"`
	ProjectName          types.String `tfsdk:"project_name"`
	AutoSync             types.Bool   `tfsdk:"auto_sync"`
	SyncInterval         types.Int64  `tfsdk:"sync_interval"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	PreDeployScriptPath  types.String `tfsdk:"pre_deploy_script_path"`
	PreDeployRunnerImage types.String `tfsdk:"pre_deploy_runner_image"`
	PreDeployEnv         types.String `tfsdk:"pre_deploy_env"`
	PreDeployExtraMounts types.String `tfsdk:"pre_deploy_extra_mounts"`
	PreDeployTimeoutSec  types.Int64  `tfsdk:"pre_deploy_timeout_sec"`
	PreDeployNetworkMode types.String `tfsdk:"pre_deploy_network_mode"`
	EnvironmentVariables types.Map    `tfsdk:"environment_variables"`
	ProjectID            types.String `tfsdk:"project_id"`
	LastSyncAt           types.String `tfsdk:"last_sync_at"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

func (d *GitOpsSyncDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config gitOpsSyncDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sync, err := d.client.GetGitOpsSync(ctx, config.EnvironmentID.ValueString(), config.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("gitops sync not found", "No gitops sync with id: "+config.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read gitops sync", err.Error())
		return
	}

	state := gitOpsSyncDataSourceModel{
		EnvironmentID: config.EnvironmentID,
		ID:            types.StringValue(sync.ID),
		Name:          types.StringValue(sync.Name),
		RepositoryID:  types.StringValue(sync.RepositoryID),
		Branch:        types.StringValue(sync.Branch),
		ComposePath:   types.StringValue(sync.ComposePath),
		Enabled:       types.BoolValue(sync.Enabled),
		AutoSync:      types.BoolValue(sync.AutoSync),
		CreatedAt:     types.StringValue(sync.CreatedAt),
		UpdatedAt:     types.StringValue(sync.UpdatedAt),
	}

	if sync.ProjectName != "" {
		state.ProjectName = types.StringValue(sync.ProjectName)
	} else {
		state.ProjectName = types.StringNull()
	}
	if sync.SyncInterval > 0 {
		state.SyncInterval = types.Int64Value(int64(sync.SyncInterval))
	} else {
		state.SyncInterval = types.Int64Null()
	}
	if sync.PreDeployScriptPath != nil && *sync.PreDeployScriptPath != "" {
		state.PreDeployScriptPath = types.StringValue(*sync.PreDeployScriptPath)
	} else {
		state.PreDeployScriptPath = types.StringNull()
	}
	if sync.PreDeployRunnerImage != nil && *sync.PreDeployRunnerImage != "" {
		state.PreDeployRunnerImage = types.StringValue(*sync.PreDeployRunnerImage)
	} else {
		state.PreDeployRunnerImage = types.StringNull()
	}
	if sync.PreDeployEnv != nil && *sync.PreDeployEnv != "" {
		state.PreDeployEnv = types.StringValue(*sync.PreDeployEnv)
	} else {
		state.PreDeployEnv = types.StringNull()
	}
	if sync.PreDeployExtraMounts != nil && *sync.PreDeployExtraMounts != "" {
		state.PreDeployExtraMounts = types.StringValue(*sync.PreDeployExtraMounts)
	} else {
		state.PreDeployExtraMounts = types.StringNull()
	}
	// These two always carry server-side defaults (60, "none"); report them
	// as-is since a data source only reflects what the API returns.
	state.PreDeployTimeoutSec = types.Int64Value(sync.PreDeployTimeoutSec)
	state.PreDeployNetworkMode = nullableString(sync.PreDeployNetworkMode)
	if sync.ProjectID != nil && *sync.ProjectID != "" {
		state.ProjectID = types.StringValue(*sync.ProjectID)
	} else {
		state.ProjectID = types.StringNull()
	}
	if sync.LastSyncAt != nil && *sync.LastSyncAt != "" {
		state.LastSyncAt = types.StringValue(*sync.LastSyncAt)
	} else {
		state.LastSyncAt = types.StringNull()
	}

	// Fetch environment variables from the project if available
	if sync.ProjectID != nil && *sync.ProjectID != "" {
		project, projErr := d.client.GetProject(ctx, config.EnvironmentID.ValueString(), *sync.ProjectID)
		if projErr == nil && project.EnvContent != nil {
			envMap, convErr := envContentToMap(ctx, *project.EnvContent)
			if convErr == nil {
				state.EnvironmentVariables = envMap
			} else {
				state.EnvironmentVariables = types.MapNull(types.StringType)
			}
		} else {
			state.EnvironmentVariables = types.MapNull(types.StringType)
		}
	} else {
		state.EnvironmentVariables = types.MapNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
