package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DeploymentSnippetDataSource{}

type DeploymentSnippetDataSource struct{ client *sdkclient.Client }

func NewDeploymentSnippetDataSource() datasource.DataSource { return &DeploymentSnippetDataSource{} }

func (d *DeploymentSnippetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_snippets"
}

func (d *DeploymentSnippetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"environment_id": schema.StringAttribute{Required: true},
		"docker_run":     schema.StringAttribute{Computed: true},
		"docker_compose": schema.StringAttribute{Computed: true},
	}}
}

func (d *DeploymentSnippetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type deploymentSnippetModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	DockerRun     types.String `tfsdk:"docker_run"`
	DockerCompose types.String `tfsdk:"docker_compose"`
}

func (d *DeploymentSnippetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state deploymentSnippetModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := d.client.GetDeploymentSnippet(ctx, state.EnvironmentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to get deployment snippets", err.Error())
		return
	}
	state.DockerRun = types.StringValue(out.DockerRun)
	state.DockerCompose = types.StringValue(out.DockerCompose)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
