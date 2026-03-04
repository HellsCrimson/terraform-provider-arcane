package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TemplateVariablesDataSource{}

type TemplateVariablesDataSource struct{ client *sdkclient.Client }

func NewTemplateVariablesDataSource() datasource.DataSource { return &TemplateVariablesDataSource{} }

func (d *TemplateVariablesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template_variables"
}

func (d *TemplateVariablesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"variables": schema.MapAttribute{Computed: true, ElementType: types.StringType},
		"data_json": schema.StringAttribute{Computed: true},
	}}
}

func (d *TemplateVariablesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type templateVariablesModel struct {
	Variables types.Map    `tfsdk:"variables"`
	DataJSON  types.String `tfsdk:"data_json"`
}

func (d *TemplateVariablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state templateVariablesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, err := d.client.GetTemplateVariables(ctx)
	if err != nil {
		resp.Diagnostics.AddError("failed to read template variables", err.Error())
		return
	}
	m := map[string]string{}
	for _, v := range items {
		m[v.Key] = v.Value
	}
	mv, diags := types.MapValueFrom(ctx, types.StringType, m)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Variables = mv
	state.DataJSON = types.StringValue(mustJSON(items))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
