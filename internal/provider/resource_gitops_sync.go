package provider

import (
	"context"
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
			"enabled": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Whether the sync is enabled",
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

func (r *GitOpsSyncResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type gitOpsSyncModel struct {
	ID             types.String `tfsdk:"id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
	Name           types.String `tfsdk:"name"`
	RepositoryID   types.String `tfsdk:"repository_id"`
	Branch         types.String `tfsdk:"branch"`
	ComposePath    types.String `tfsdk:"compose_path"`
	ProjectName    types.String `tfsdk:"project_name"`
	AutoSync       types.Bool   `tfsdk:"auto_sync"`
	SyncInterval   types.Int64  `tfsdk:"sync_interval"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	ProjectID      types.String `tfsdk:"project_id"`
	LastSyncAt     types.String `tfsdk:"last_sync_at"`
	LastSyncCommit types.String `tfsdk:"last_sync_commit"`
	LastSyncStatus types.String `tfsdk:"last_sync_status"`
	LastSyncError  types.String `tfsdk:"last_sync_error"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
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
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		body.Enabled = &v
	}

	sync, err := r.client.CreateGitOpsSync(ctx, plan.EnvironmentID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("create gitops sync failed", err.Error())
		return
	}

	state := gitOpsSyncModel{
		ID:            types.StringValue(sync.ID),
		EnvironmentID: types.StringValue(sync.EnvironmentID),
		Name:          types.StringValue(sync.Name),
		RepositoryID:  types.StringValue(sync.RepositoryID),
		Branch:        types.StringValue(sync.Branch),
		ComposePath:   types.StringValue(sync.ComposePath),
		ProjectName:   types.StringValue(sync.ProjectName),
		AutoSync:      types.BoolValue(sync.AutoSync),
		SyncInterval:  types.Int64Value(sync.SyncInterval),
		Enabled:       types.BoolValue(sync.Enabled),
		CreatedAt:     types.StringValue(sync.CreatedAt),
		UpdatedAt:     types.StringValue(sync.UpdatedAt),
	}

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
		if strings.Contains(strings.ToLower(err.Error()), "404") {
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
	state.ProjectName = types.StringValue(sync.ProjectName)
	state.AutoSync = types.BoolValue(sync.AutoSync)
	state.SyncInterval = types.Int64Value(sync.SyncInterval)
	state.Enabled = types.BoolValue(sync.Enabled)
	state.UpdatedAt = types.StringValue(sync.UpdatedAt)

	if sync.ProjectID != nil {
		state.ProjectID = types.StringValue(*sync.ProjectID)
	} else {
		state.ProjectID = types.StringNull()
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
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		body.Enabled = &v
	}

	sync, err := r.client.UpdateGitOpsSync(ctx, state.EnvironmentID.ValueString(), state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("update gitops sync failed", err.Error())
		return
	}

	state.Name = types.StringValue(sync.Name)
	state.RepositoryID = types.StringValue(sync.RepositoryID)
	state.Branch = types.StringValue(sync.Branch)
	state.ComposePath = types.StringValue(sync.ComposePath)
	state.ProjectName = types.StringValue(sync.ProjectName)
	state.AutoSync = types.BoolValue(sync.AutoSync)
	state.SyncInterval = types.Int64Value(sync.SyncInterval)
	state.Enabled = types.BoolValue(sync.Enabled)
	state.UpdatedAt = types.StringValue(sync.UpdatedAt)

	if sync.ProjectID != nil {
		state.ProjectID = types.StringValue(*sync.ProjectID)
	} else {
		state.ProjectID = types.StringNull()
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

	if err := r.client.DeleteGitOpsSync(ctx, state.EnvironmentID.ValueString(), state.ID.ValueString()); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete gitops sync failed", err.Error())
	}
}

func (r *GitOpsSyncResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
