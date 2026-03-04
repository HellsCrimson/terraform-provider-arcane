package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &PublicSettingsDataSource{}

type PublicSettingsDataSource struct{ client *sdkclient.Client }

func NewPublicSettingsDataSource() datasource.DataSource { return &PublicSettingsDataSource{} }

func (d *PublicSettingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_public_settings"
}

func (d *PublicSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"environment_id": schema.StringAttribute{Required: true},
		"settings":       schema.MapAttribute{Computed: true, ElementType: types.StringType},
	}}
}

func (d *PublicSettingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type publicSettingsModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	Settings      types.Map    `tfsdk:"settings"`
}

func (d *PublicSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state publicSettingsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	settings, err := d.client.GetPublicSettings(ctx, state.EnvironmentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to read public settings", err.Error())
		return
	}
	mv, diags := types.MapValueFrom(ctx, types.StringType, settings)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Settings = mv
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
