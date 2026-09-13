package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SwarmConfigDataSource{}

type SwarmConfigDataSource struct {
	client *sdkclient.Client
}

func NewSwarmConfigDataSource() datasource.DataSource {
	return &SwarmConfigDataSource{}
}

func (d *SwarmConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_swarm_config"
}

func (d *SwarmConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane Docker Swarm config",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{Required: true, Description: "Environment ID"},
			"id":             schema.StringAttribute{Required: true, Description: "Swarm config ID"},
			"name":           schema.StringAttribute{Computed: true, Description: "Config name"},
			"labels": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Config labels",
			},
			"version_index": schema.Int64Attribute{Computed: true, Description: "Swarm object version index"},
			"created_at":    schema.StringAttribute{Computed: true, Description: "Creation timestamp"},
			"updated_at":    schema.StringAttribute{Computed: true, Description: "Last update timestamp"},
		},
	}
}

func (d *SwarmConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type swarmConfigDataSourceModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Labels        types.Map    `tfsdk:"labels"`
	VersionIndex  types.Int64  `tfsdk:"version_index"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (d *SwarmConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config swarmConfigDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := d.client.GetSwarmConfig(ctx, config.EnvironmentID.ValueString(), config.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("swarm config not found", "No swarm config with id: "+config.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read swarm config", err.Error())
		return
	}

	state := swarmConfigDataSourceModel{
		EnvironmentID: config.EnvironmentID,
		ID:            types.StringValue(cfg.ID),
		Name:          types.StringValue(cfg.Spec.Name),
		Labels:        stringMapToMap(ctx, cfg.Spec.Labels),
		VersionIndex:  types.Int64Value(cfg.Version.Index),
		CreatedAt:     types.StringValue(cfg.CreatedAt),
		UpdatedAt:     types.StringValue(cfg.UpdatedAt),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
