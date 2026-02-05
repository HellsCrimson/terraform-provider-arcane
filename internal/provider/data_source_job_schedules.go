package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &JobSchedulesDataSource{}

type JobSchedulesDataSource struct {
	client *sdkclient.Client
}

func NewJobSchedulesDataSource() datasource.DataSource {
	return &JobSchedulesDataSource{}
}

func (d *JobSchedulesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job_schedules"
}

func (d *JobSchedulesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading Arcane job schedules configuration",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource ID (same as environment_id)",
			},
			"analytics_heartbeat_interval": schema.StringAttribute{
				Computed:    true,
				Description: "Analytics heartbeat cron expression",
			},
			"auto_update_interval": schema.StringAttribute{
				Computed:    true,
				Description: "Auto-update check cron expression",
			},
			"environment_health_interval": schema.StringAttribute{
				Computed:    true,
				Description: "Environment health check cron expression",
			},
			"event_cleanup_interval": schema.StringAttribute{
				Computed:    true,
				Description: "Event cleanup cron expression",
			},
			"gitops_sync_interval": schema.StringAttribute{
				Computed:    true,
				Description: "GitOps sync cron expression",
			},
			"polling_interval": schema.StringAttribute{
				Computed:    true,
				Description: "Polling interval cron expression",
			},
			"scheduled_prune_interval": schema.StringAttribute{
				Computed:    true,
				Description: "Scheduled prune cron expression",
			},
		},
	}
}

func (d *JobSchedulesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type jobSchedulesDataSourceModel struct {
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

func (d *JobSchedulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config jobSchedulesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedules, err := d.client.GetJobSchedules(ctx, config.EnvironmentID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("job schedules not found", "No job schedules for environment: "+config.EnvironmentID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read job schedules", err.Error())
		return
	}

	state := jobSchedulesDataSourceModel{
		ID:                         config.EnvironmentID,
		EnvironmentID:              config.EnvironmentID,
		AnalyticsHeartbeatInterval: types.StringValue(schedules.AnalyticsHeartbeatInterval),
		AutoUpdateInterval:         types.StringValue(schedules.AutoUpdateInterval),
		EnvironmentHealthInterval:  types.StringValue(schedules.EnvironmentHealthInterval),
		EventCleanupInterval:       types.StringValue(schedules.EventCleanupInterval),
		GitOpsSyncInterval:         types.StringValue(schedules.GitOpsSyncInterval),
		PollingInterval:            types.StringValue(schedules.PollingInterval),
		ScheduledPruneInterval:     types.StringValue(schedules.ScheduledPruneInterval),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
