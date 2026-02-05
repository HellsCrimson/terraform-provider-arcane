package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &NetworkDataSource{}

type NetworkDataSource struct {
	client *sdkclient.Client
}

func NewNetworkDataSource() datasource.DataSource {
	return &NetworkDataSource{}
}

func (d *NetworkDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (d *NetworkDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane network",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Network ID",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Network name",
			},
			"driver": schema.StringAttribute{
				Computed:    true,
				Description: "Network driver",
			},
			"attachable": schema.BoolAttribute{
				Computed:    true,
				Description: "Allow manual container attachment",
			},
			"internal": schema.BoolAttribute{
				Computed:    true,
				Description: "Restrict external access",
			},
			"enable_ipv4": schema.BoolAttribute{
				Computed:    true,
				Description: "IPv4 enabled",
			},
			"enable_ipv6": schema.BoolAttribute{
				Computed:    true,
				Description: "IPv6 enabled",
			},
			"scope": schema.StringAttribute{
				Computed:    true,
				Description: "Network scope",
			},
			"created": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
			"labels": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Network labels",
			},
			"options": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Network options",
			},
		},
	}
}

func (d *NetworkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type networkDataSourceModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Driver        types.String `tfsdk:"driver"`
	Attachable    types.Bool   `tfsdk:"attachable"`
	Internal      types.Bool   `tfsdk:"internal"`
	EnableIPv4    types.Bool   `tfsdk:"enable_ipv4"`
	EnableIPv6    types.Bool   `tfsdk:"enable_ipv6"`
	Scope         types.String `tfsdk:"scope"`
	Created       types.String `tfsdk:"created"`
	Labels        types.Map    `tfsdk:"labels"`
	Options       types.Map    `tfsdk:"options"`
}

func (d *NetworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config networkDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	network, err := d.client.GetNetwork(ctx, config.EnvironmentID.ValueString(), config.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("network not found", "No network with id: "+config.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read network", err.Error())
		return
	}

	state := networkDataSourceModel{
		EnvironmentID: config.EnvironmentID,
		ID:            types.StringValue(network.ID),
		Name:          types.StringValue(network.Name),
		Driver:        types.StringValue(network.Driver),
		Attachable:    types.BoolValue(network.Attachable),
		Internal:      types.BoolValue(network.Internal),
		EnableIPv4:    types.BoolValue(network.EnableIPv4),
		EnableIPv6:    types.BoolValue(network.EnableIPv6),
		Scope:         types.StringValue(network.Scope),
		Created:       types.StringValue(network.Created),
	}

	state.Labels = stringMapToMap(ctx, network.Labels)
	state.Options = stringMapToMap(ctx, network.Options)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
