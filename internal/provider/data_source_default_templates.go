package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DefaultTemplatesDataSource{}

type DefaultTemplatesDataSource struct{ client *sdkclient.Client }

func NewDefaultTemplatesDataSource() datasource.DataSource { return &DefaultTemplatesDataSource{} }

func (d *DefaultTemplatesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_default_templates"
}

func (d *DefaultTemplatesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"compose_template": schema.StringAttribute{Computed: true},
		"env_template":     schema.StringAttribute{Computed: true},
	}}
}

func (d *DefaultTemplatesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type defaultTemplatesModel struct {
	ComposeTemplate types.String `tfsdk:"compose_template"`
	EnvTemplate     types.String `tfsdk:"env_template"`
}

func (d *DefaultTemplatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state defaultTemplatesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := d.client.GetDefaultTemplates(ctx)
	if err != nil {
		resp.Diagnostics.AddError("failed to read default templates", err.Error())
		return
	}
	state.ComposeTemplate = types.StringValue(out.ComposeTemplate)
	state.EnvTemplate = types.StringValue(out.EnvTemplate)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
