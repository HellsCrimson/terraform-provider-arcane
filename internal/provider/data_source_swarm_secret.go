package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SwarmSecretDataSource{}

type SwarmSecretDataSource struct {
	client *sdkclient.Client
}

func NewSwarmSecretDataSource() datasource.DataSource {
	return &SwarmSecretDataSource{}
}

func (d *SwarmSecretDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_swarm_secret"
}

func (d *SwarmSecretDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane Docker Swarm secret",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{Required: true, Description: "Environment ID"},
			"id":             schema.StringAttribute{Required: true, Description: "Swarm secret ID"},
			"name":           schema.StringAttribute{Computed: true, Description: "Secret name"},
			"labels": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Secret labels",
			},
			"version_index": schema.Int64Attribute{Computed: true, Description: "Swarm object version index"},
			"created_at":    schema.StringAttribute{Computed: true, Description: "Creation timestamp"},
			"updated_at":    schema.StringAttribute{Computed: true, Description: "Last update timestamp"},
		},
	}
}

func (d *SwarmSecretDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type swarmSecretDataSourceModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Labels        types.Map    `tfsdk:"labels"`
	VersionIndex  types.Int64  `tfsdk:"version_index"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (d *SwarmSecretDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config swarmSecretDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := d.client.GetSwarmSecret(ctx, config.EnvironmentID.ValueString(), config.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("swarm secret not found", "No swarm secret with id: "+config.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read swarm secret", err.Error())
		return
	}

	state := swarmSecretDataSourceModel{
		EnvironmentID: config.EnvironmentID,
		ID:            types.StringValue(secret.ID),
		Name:          types.StringValue(secret.Spec.Name),
		Labels:        stringMapToMap(ctx, secret.Spec.Labels),
		VersionIndex:  types.Int64Value(secret.Version.Index),
		CreatedAt:     types.StringValue(secret.CreatedAt),
		UpdatedAt:     types.StringValue(secret.UpdatedAt),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
