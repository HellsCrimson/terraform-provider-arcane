package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ContainerDataSource{}

type ContainerDataSource struct {
	client *sdkclient.Client
}

func NewContainerDataSource() datasource.DataSource {
	return &ContainerDataSource{}
}

func (d *ContainerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container"
}

func (d *ContainerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane container",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Container ID",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Container name",
			},
			"image": schema.StringAttribute{
				Computed:    true,
				Description: "Container image",
			},
			"created": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Container status",
			},
		},
	}
}

func (d *ContainerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type containerDataSourceModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Image         types.String `tfsdk:"image"`
	Created       types.String `tfsdk:"created"`
	Status        types.String `tfsdk:"status"`
}

func (d *ContainerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config containerDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	container, err := d.client.GetContainer(ctx, config.EnvironmentID.ValueString(), config.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("container not found", "No container with id: "+config.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read container", err.Error())
		return
	}

	state := containerDataSourceModel{
		EnvironmentID: config.EnvironmentID,
		ID:            types.StringValue(container.ID),
		Name:          types.StringValue(container.Name),
		Image:         types.StringValue(container.Image),
		Created:       types.StringValue(container.Created),
		Status:        types.StringValue(container.Status),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
