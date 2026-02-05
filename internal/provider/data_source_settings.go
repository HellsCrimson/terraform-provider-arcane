package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SettingsDataSource{}

type SettingsDataSource struct {
	client *sdkclient.Client
}

func NewSettingsDataSource() datasource.DataSource {
	return &SettingsDataSource{}
}

func (d *SettingsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_settings"
}

func (d *SettingsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading Arcane environment settings as a key-value map",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
			},
			"settings": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "All environment settings as a key-value map",
			},
		},
	}
}

func (d *SettingsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type settingsDataSourceModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	Settings      types.Map    `tfsdk:"settings"`
}

func (d *SettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config settingsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := d.client.GetSettings(ctx, config.EnvironmentID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("settings not found", "No settings for environment: "+config.EnvironmentID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read settings", err.Error())
		return
	}

	// Convert settings to map
	settingsMap, diags := types.MapValueFrom(ctx, types.StringType, settings)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := settingsDataSourceModel{
		EnvironmentID: config.EnvironmentID,
		Settings:      settingsMap,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
