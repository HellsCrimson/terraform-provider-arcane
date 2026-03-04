package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SettingsCategoriesDataSource{}

type SettingsCategoriesDataSource struct{ client *sdkclient.Client }

func NewSettingsCategoriesDataSource() datasource.DataSource { return &SettingsCategoriesDataSource{} }

func (d *SettingsCategoriesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_settings_categories"
}

func (d *SettingsCategoriesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"total_count": schema.Int64Attribute{Computed: true},
		"data_json":   schema.StringAttribute{Computed: true},
	}}
}

func (d *SettingsCategoriesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

func (d *SettingsCategoriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state categoriesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, err := d.client.GetSettingsCategories(ctx)
	if err != nil {
		resp.Diagnostics.AddError("failed to read settings categories", err.Error())
		return
	}
	state.TotalCount = types.Int64Value(int64(len(items)))
	state.DataJSON = types.StringValue(mustJSON(items))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
