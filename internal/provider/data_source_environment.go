package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &EnvironmentDataSource{}

type EnvironmentDataSource struct {
	client *sdkclient.Client
}

func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

func (d *EnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *EnvironmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane environment",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Environment display name",
			},
			"api_url": schema.StringAttribute{
				Computed:    true,
				Description: "Agent API URL",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Environment status",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the environment is enabled",
			},
			"api_key": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Environment API key",
			},
		},
	}
}

func (d *EnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type environmentDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	APIURL  types.String `tfsdk:"api_url"`
	Status  types.String `tfsdk:"status"`
	Enabled types.Bool   `tfsdk:"enabled"`
	APIKey  types.String `tfsdk:"api_key"`
}

func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config environmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	env, err := d.client.GetEnvironment(ctx, config.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("environment not found", "No environment with id: "+config.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read environment", err.Error())
		return
	}

	state := environmentDataSourceModel{
		ID:      types.StringValue(env.ID),
		Name:    types.StringValue(env.Name),
		APIURL:  types.StringValue(env.APIURL),
		Status:  types.StringValue(env.Status),
		Enabled: types.BoolValue(env.Enabled),
	}
	if env.APIKey != "" {
		state.APIKey = types.StringValue(env.APIKey)
	} else {
		state.APIKey = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
