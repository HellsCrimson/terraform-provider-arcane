package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ProjectSectionDataSource{}

type ProjectSectionDataSource struct {
	client   *sdkclient.Client
	typeName string
	section  string
	desc     string
}

func NewProjectComposeDataSource() datasource.DataSource {
	return &ProjectSectionDataSource{
		typeName: "project_compose",
		section:  "compose",
		desc:     "Reads compose content, includes, and service config details for a project.",
	}
}

func NewProjectFilesDataSource() datasource.DataSource {
	return &ProjectSectionDataSource{
		typeName: "project_files",
		section:  "files",
		desc:     "Reads directory file details for a project.",
	}
}

func NewProjectRuntimeDataSource() datasource.DataSource {
	return &ProjectSectionDataSource{
		typeName: "project_runtime",
		section:  "runtime",
		desc:     "Reads runtime service state for a project.",
	}
}

func NewProjectUpdatesDataSource() datasource.DataSource {
	return &ProjectSectionDataSource{
		typeName: "project_updates",
		section:  "updates",
		desc:     "Reads image update summary details for a project.",
	}
}

func (d *ProjectSectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.typeName
}

func (d *ProjectSectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: d.desc,
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{Required: true},
			"project_id":     schema.StringAttribute{Required: true},
			"id":             schema.StringAttribute{Computed: true},
			"details_json": schema.StringAttribute{
				Computed:    true,
				Description: "Raw ProjectDetails response data as JSON.",
			},
		},
	}
}

func (d *ProjectSectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type projectSectionModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	ProjectID     types.String `tfsdk:"project_id"`
	ID            types.String `tfsdk:"id"`
	DetailsJSON   types.String `tfsdk:"details_json"`
}

func (d *ProjectSectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectSectionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := state.EnvironmentID.ValueString()
	projectID := state.ProjectID.ValueString()
	data, err := d.client.GetProjectSectionData(ctx, envID, projectID, d.section)
	if err != nil {
		resp.Diagnostics.AddError("failed to read project "+d.section, err.Error())
		return
	}
	state.ID = types.StringValue(envID + ":" + projectID + ":" + d.section)
	state.DetailsJSON = types.StringValue(mustJSON(data))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
