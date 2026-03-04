package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ProjectIncludesDataSource{}

type ProjectIncludesDataSource struct{ client *sdkclient.Client }

func NewProjectIncludesDataSource() datasource.DataSource { return &ProjectIncludesDataSource{} }

func (d *ProjectIncludesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_includes"
}

func (d *ProjectIncludesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"environment_id": schema.StringAttribute{Required: true},
		"project_id":     schema.StringAttribute{Required: true},
		"count":          schema.Int64Attribute{Computed: true},
		"includes_json":  schema.StringAttribute{Computed: true},
	}}
}

func (d *ProjectIncludesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type projectIncludesModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	ProjectID     types.String `tfsdk:"project_id"`
	Count         types.Int64  `tfsdk:"count"`
	IncludesJSON  types.String `tfsdk:"includes_json"`
}

func (d *ProjectIncludesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectIncludesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := d.client.GetProject(ctx, state.EnvironmentID.ValueString(), state.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to read project", err.Error())
		return
	}
	state.Count = types.Int64Value(int64(len(out.IncludeFiles)))
	state.IncludesJSON = types.StringValue(mustJSON(out.IncludeFiles))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
