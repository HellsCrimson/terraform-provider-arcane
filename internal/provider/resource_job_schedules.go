package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &JobSchedulesResource{}
var _ resource.ResourceWithImportState = &JobSchedulesResource{}

type JobSchedulesResource struct {
	client *sdkclient.Client
}

func NewJobSchedulesResource() resource.Resource {
	return &JobSchedulesResource{}
}

func (r *JobSchedulesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job_schedules"
}

func (r *JobSchedulesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manages cron schedules for automated background jobs in an environment. All intervals use cron format.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:      true,
				Description:   "Resource ID (same as environment_id)",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id": resourceschema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
			},
			"analytics_heartbeat_interval": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Cron expression for analytics heartbeat (e.g., '0 */5 * * * *' for every 5 minutes)",
			},
			"auto_update_interval": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Cron expression for auto-update checks (e.g., '0 0 2 * * *' for daily at 2 AM)",
			},
			"environment_health_interval": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Cron expression for environment health checks (e.g., '0 */1 * * * *' for every minute)",
			},
			"event_cleanup_interval": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Cron expression for event log cleanup (e.g., '0 0 3 * * *' for daily at 3 AM)",
			},
			"gitops_sync_interval": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Cron expression for GitOps sync checks (e.g., '0 */10 * * * *' for every 10 minutes)",
			},
			"polling_interval": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Cron expression for general polling operations (e.g., '0 */5 * * * *' for every 5 minutes)",
			},
			"scheduled_prune_interval": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Cron expression for scheduled pruning of Docker resources (e.g., '0 0 1 * * 0' for weekly on Sunday at 1 AM)",
			},
		},
	}
}

func (r *JobSchedulesResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type jobSchedulesModel struct {
	ID                         types.String `tfsdk:"id"`
	EnvironmentID              types.String `tfsdk:"environment_id"`
	AnalyticsHeartbeatInterval types.String `tfsdk:"analytics_heartbeat_interval"`
	AutoUpdateInterval         types.String `tfsdk:"auto_update_interval"`
	EnvironmentHealthInterval  types.String `tfsdk:"environment_health_interval"`
	EventCleanupInterval       types.String `tfsdk:"event_cleanup_interval"`
	GitOpsSyncInterval         types.String `tfsdk:"gitops_sync_interval"`
	PollingInterval            types.String `tfsdk:"polling_interval"`
	ScheduledPruneInterval     types.String `tfsdk:"scheduled_prune_interval"`
}

func (r *JobSchedulesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan jobSchedulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envID := plan.EnvironmentID.ValueString()
	body := buildJobSchedulesRequest(plan)

	if _, err := r.client.UpdateJobSchedules(ctx, envID, body); err != nil {
		resp.Diagnostics.AddError("update job schedules failed", err.Error())
		return
	}

	config, err := r.client.GetJobSchedules(ctx, envID)
	if err != nil {
		resp.Diagnostics.AddError("read job schedules failed", err.Error())
		return
	}

	state := plan
	state.ID = types.StringValue(envID)
	state.EnvironmentID = types.StringValue(envID)
	applyJobSchedulesConfig(&state, config)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *JobSchedulesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state jobSchedulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envID := state.EnvironmentID.ValueString()
	_, err := r.client.GetJobSchedules(ctx, envID)
	if err != nil {
		resp.Diagnostics.AddError("read job schedules failed", err.Error())
		return
	}

	state.ID = types.StringValue(envID)
	// Only update fields that were set by user (preserve plan values)
	// The actual intervals are managed by the API

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *JobSchedulesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan jobSchedulesModel
	var state jobSchedulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envID := state.EnvironmentID.ValueString()
	body := buildJobSchedulesRequest(plan)

	if _, err := r.client.UpdateJobSchedules(ctx, envID, body); err != nil {
		resp.Diagnostics.AddError("update job schedules failed", err.Error())
		return
	}

	config, err := r.client.GetJobSchedules(ctx, envID)
	if err != nil {
		resp.Diagnostics.AddError("read job schedules failed", err.Error())
		return
	}

	state = plan
	state.ID = types.StringValue(envID)
	state.EnvironmentID = types.StringValue(envID)
	applyJobSchedulesConfig(&state, config)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *JobSchedulesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Job schedules cannot be deleted, only reset to defaults
	// Just remove from state
}

func (r *JobSchedulesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by environment ID
	resource.ImportStatePassthroughID(ctx, path.Root("environment_id"), req, resp)
}

// Helper functions
func buildJobSchedulesRequest(m jobSchedulesModel) sdkclient.UpdateJobSchedulesRequest {
	req := sdkclient.UpdateJobSchedulesRequest{}

	if !m.AnalyticsHeartbeatInterval.IsNull() && !m.AnalyticsHeartbeatInterval.IsUnknown() {
		v := m.AnalyticsHeartbeatInterval.ValueString()
		req.AnalyticsHeartbeatInterval = &v
	}
	if !m.AutoUpdateInterval.IsNull() && !m.AutoUpdateInterval.IsUnknown() {
		v := m.AutoUpdateInterval.ValueString()
		req.AutoUpdateInterval = &v
	}
	if !m.EnvironmentHealthInterval.IsNull() && !m.EnvironmentHealthInterval.IsUnknown() {
		v := m.EnvironmentHealthInterval.ValueString()
		req.EnvironmentHealthInterval = &v
	}
	if !m.EventCleanupInterval.IsNull() && !m.EventCleanupInterval.IsUnknown() {
		v := m.EventCleanupInterval.ValueString()
		req.EventCleanupInterval = &v
	}
	if !m.GitOpsSyncInterval.IsNull() && !m.GitOpsSyncInterval.IsUnknown() {
		v := m.GitOpsSyncInterval.ValueString()
		req.GitOpsSyncInterval = &v
	}
	if !m.PollingInterval.IsNull() && !m.PollingInterval.IsUnknown() {
		v := m.PollingInterval.ValueString()
		req.PollingInterval = &v
	}
	if !m.ScheduledPruneInterval.IsNull() && !m.ScheduledPruneInterval.IsUnknown() {
		v := m.ScheduledPruneInterval.ValueString()
		req.ScheduledPruneInterval = &v
	}

	return req
}

func applyJobSchedulesConfig(m *jobSchedulesModel, config *sdkclient.JobSchedulesConfig) {
	// Only update if value was set in plan, otherwise leave as-is
	// This allows partial updates
	if !m.AnalyticsHeartbeatInterval.IsNull() {
		m.AnalyticsHeartbeatInterval = types.StringValue(config.AnalyticsHeartbeatInterval)
	}
	if !m.AutoUpdateInterval.IsNull() {
		m.AutoUpdateInterval = types.StringValue(config.AutoUpdateInterval)
	}
	if !m.EnvironmentHealthInterval.IsNull() {
		m.EnvironmentHealthInterval = types.StringValue(config.EnvironmentHealthInterval)
	}
	if !m.EventCleanupInterval.IsNull() {
		m.EventCleanupInterval = types.StringValue(config.EventCleanupInterval)
	}
	if !m.GitOpsSyncInterval.IsNull() {
		m.GitOpsSyncInterval = types.StringValue(config.GitOpsSyncInterval)
	}
	if !m.PollingInterval.IsNull() {
		m.PollingInterval = types.StringValue(config.PollingInterval)
	}
	if !m.ScheduledPruneInterval.IsNull() {
		m.ScheduledPruneInterval = types.StringValue(config.ScheduledPruneInterval)
	}
}
