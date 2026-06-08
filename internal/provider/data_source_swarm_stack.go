package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SwarmStackDataSource{}

type SwarmStackDataSource struct {
	client *sdkclient.Client
}

func NewSwarmStackDataSource() datasource.DataSource {
	return &SwarmStackDataSource{}
}

func (d *SwarmStackDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_swarm_stack"
}

func (d *SwarmStackDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane Docker Swarm stack",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{Required: true, Description: "Environment ID"},
			"id":             schema.StringAttribute{Required: true, Description: "Stack name"},
			"name":           schema.StringAttribute{Computed: true, Description: "Stack name"},
			"compose_content": schema.StringAttribute{
				Computed:    true,
				Description: "Docker Compose content for the stack",
			},
			"env_content": schema.StringAttribute{Computed: true, Description: ".env content for the stack"},
			"namespace":   schema.StringAttribute{Computed: true, Description: "Docker namespace for the stack"},
			"services":    schema.Int64Attribute{Computed: true, Description: "Number of services in the stack"},
			"created_at":  schema.StringAttribute{Computed: true, Description: "Creation timestamp"},
			"updated_at":  schema.StringAttribute{Computed: true, Description: "Last update timestamp"},
		},
	}
}

func (d *SwarmStackDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type swarmStackDataSourceModel struct {
	EnvironmentID  types.String `tfsdk:"environment_id"`
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	ComposeContent types.String `tfsdk:"compose_content"`
	EnvContent     types.String `tfsdk:"env_content"`
	Namespace      types.String `tfsdk:"namespace"`
	Services       types.Int64  `tfsdk:"services"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (d *SwarmStackDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config swarmStackDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envID := config.EnvironmentID.ValueString()
	stackName := config.ID.ValueString()

	inspect, err := d.client.GetSwarmStack(ctx, envID, stackName)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("swarm stack not found", "No swarm stack with name: "+stackName)
			return
		}
		resp.Diagnostics.AddError("failed to read swarm stack", err.Error())
		return
	}

	source, err := d.client.GetSwarmStackSource(ctx, envID, stackName)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("swarm stack source not found", "No source for swarm stack with name: "+stackName)
			return
		}
		resp.Diagnostics.AddError("failed to read swarm stack source", err.Error())
		return
	}

	state := swarmStackDataSourceModel{
		EnvironmentID:  config.EnvironmentID,
		ID:             types.StringValue(inspect.Name),
		Name:           types.StringValue(inspect.Name),
		ComposeContent: types.StringValue(source.ComposeContent),
		Namespace:      types.StringValue(inspect.Namespace),
		Services:       types.Int64Value(inspect.Services),
		CreatedAt:      types.StringValue(inspect.CreatedAt),
		UpdatedAt:      types.StringValue(inspect.UpdatedAt),
	}
	if source.EnvContent != "" {
		state.EnvContent = types.StringValue(source.EnvContent)
	} else {
		state.EnvContent = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
