package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ProjectPathDataSource{}

type ProjectPathDataSource struct {
	client *sdkclient.Client
}

func NewProjectPathDataSource() datasource.DataSource {
	return &ProjectPathDataSource{}
}

func (d *ProjectPathDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_path"
}

func (d *ProjectPathDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane project (project_path resource)",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Project ID",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Project name",
			},
			"compose_content": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Docker Compose content",
			},
			"env_content": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Environment variables content",
			},
			"path": schema.StringAttribute{
				Computed:    true,
				Description: "Project path on the environment",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Project status",
			},
			"service_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of services",
			},
			"running_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of running services",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
			},
		},
	}
}

func (d *ProjectPathDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type projectPathDataSourceModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Compose       types.String `tfsdk:"compose_content"`
	Env           types.String `tfsdk:"env_content"`
	Path          types.String `tfsdk:"path"`
	Status        types.String `tfsdk:"status"`
	ServiceCount  types.Int64  `tfsdk:"service_count"`
	RunningCount  types.Int64  `tfsdk:"running_count"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (d *ProjectPathDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config projectPathDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, err := d.client.GetProject(ctx, config.EnvironmentID.ValueString(), config.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("project not found", "No project with id: "+config.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read project", err.Error())
		return
	}

	state := projectPathDataSourceModel{
		EnvironmentID: config.EnvironmentID,
		ID:            types.StringValue(project.ID),
		Name:          types.StringValue(project.Name),
		Path:          types.StringValue(project.Path),
		Status:        types.StringValue(project.Status),
		ServiceCount:  types.Int64Value(int64(project.ServiceCount)),
		RunningCount:  types.Int64Value(int64(project.RunningCount)),
		CreatedAt:     types.StringValue(project.CreatedAt),
		UpdatedAt:     types.StringValue(project.UpdatedAt),
	}

	if project.ComposeContent != nil {
		state.Compose = types.StringValue(*project.ComposeContent)
	} else {
		state.Compose = types.StringNull()
	}
	if project.EnvContent != nil {
		state.Env = types.StringValue(*project.EnvContent)
	} else {
		state.Env = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
