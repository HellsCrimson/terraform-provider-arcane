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
	resp.Schema = schema.Schema{
		Description: "Agent deployment snippets for an environment. Since Arcane v3 every read mints a new single-use " +
			"enrollment token (embedded in the snippets), so each plan/refresh produces a fresh token. Prefer the " +
			"enrollment_token of the arcane_environment resource for a stable value.",
		Attributes: map[string]schema.Attribute{
			"environment_id":      schema.StringAttribute{Required: true},
			"docker_run":          schema.StringAttribute{Computed: true, Sensitive: true, Description: "docker run command (embeds the enrollment token)"},
			"docker_compose":      schema.StringAttribute{Computed: true, Sensitive: true, Description: "Compose snippet (embeds the enrollment token)"},
			"enrollment_token":    schema.StringAttribute{Computed: true, Sensitive: true, Description: "Single-use enrollment token minted by this read"},
			"mtls_docker_run":     schema.StringAttribute{Computed: true},
			"mtls_docker_compose": schema.StringAttribute{Computed: true},
			"mtls_host_dir_hint":  schema.StringAttribute{Computed: true},
			"mtls_files_json":     schema.StringAttribute{Computed: true, Sensitive: true},
		}}
}

func (d *DeploymentSnippetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type deploymentSnippetModel struct {
	EnvironmentID     types.String `tfsdk:"environment_id"`
	DockerRun         types.String `tfsdk:"docker_run"`
	DockerCompose     types.String `tfsdk:"docker_compose"`
	EnrollmentToken   types.String `tfsdk:"enrollment_token"`
	MTLSDockerRun     types.String `tfsdk:"mtls_docker_run"`
	MTLSDockerCompose types.String `tfsdk:"mtls_docker_compose"`
	MTLSHostDirHint   types.String `tfsdk:"mtls_host_dir_hint"`
	MTLSFilesJSON     types.String `tfsdk:"mtls_files_json"`
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
	state.EnrollmentToken = nullableString(out.EnrollmentToken)
	if out.MTLS != nil {
		state.MTLSDockerRun = types.StringValue(out.MTLS.DockerRun)
		state.MTLSDockerCompose = types.StringValue(out.MTLS.DockerCompose)
		state.MTLSHostDirHint = types.StringValue(out.MTLS.HostDirHint)
		state.MTLSFilesJSON = types.StringValue(mustJSON(out.MTLS.Files))
	} else {
		state.MTLSDockerRun = types.StringNull()
		state.MTLSDockerCompose = types.StringNull()
		state.MTLSHostDirHint = types.StringNull()
		state.MTLSFilesJSON = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
