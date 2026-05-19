package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ContainerRegistryPullUsageDataSource{}

type ContainerRegistryPullUsageDataSource struct {
	client *sdkclient.Client
}

func NewContainerRegistryPullUsageDataSource() datasource.DataSource {
	return &ContainerRegistryPullUsageDataSource{}
}

func (d *ContainerRegistryPullUsageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry_pull_usage"
}

func (d *ContainerRegistryPullUsageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads configured container registry pull usage and rate limit visibility.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true},
			"total_count": schema.Int64Attribute{Computed: true},
			"registries_json": schema.StringAttribute{
				Computed:    true,
				Description: "Registry pull usage entries as JSON.",
			},
		},
	}
}

func (d *ContainerRegistryPullUsageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type containerRegistryPullUsageModel struct {
	ID             types.String `tfsdk:"id"`
	TotalCount     types.Int64  `tfsdk:"total_count"`
	RegistriesJSON types.String `tfsdk:"registries_json"`
}

func (d *ContainerRegistryPullUsageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state containerRegistryPullUsageModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	usage, err := d.client.GetContainerRegistryPullUsage(ctx)
	if err != nil {
		resp.Diagnostics.AddError("failed to read container registry pull usage", err.Error())
		return
	}

	state.ID = types.StringValue("container-registry-pull-usage")
	state.TotalCount = types.Int64Value(int64(len(usage)))
	state.RegistriesJSON = types.StringValue(mustJSON(usage))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
