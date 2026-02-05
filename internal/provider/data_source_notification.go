package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &NotificationDataSource{}

type NotificationDataSource struct {
	client *sdkclient.Client
}

func NewNotificationDataSource() datasource.DataSource {
	return &NotificationDataSource{}
}

func (d *NotificationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification"
}

func (d *NotificationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane notification configuration",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
			},
			"provider_name": schema.StringAttribute{
				Required:    true,
				Description: "Notification provider name",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Notification ID (environment_id:provider_name)",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the notification is enabled",
			},
			"config": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Provider-specific configuration",
			},
		},
	}
}

func (d *NotificationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type notificationDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	ProviderName  types.String `tfsdk:"provider_name"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Config        types.Map    `tfsdk:"config"`
}

func (d *NotificationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config notificationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	notification, err := d.client.GetNotification(ctx, config.EnvironmentID.ValueString(), config.ProviderName.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("notification not found", "No notification for provider: "+config.ProviderName.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read notification", err.Error())
		return
	}

	state := notificationDataSourceModel{
		ID:            types.StringValue(config.EnvironmentID.ValueString() + ":" + notification.Provider),
		EnvironmentID: config.EnvironmentID,
		ProviderName:  types.StringValue(notification.Provider),
		Enabled:       types.BoolValue(notification.Enabled),
		Config:        anyMapToStringMap(ctx, notification.Config),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
