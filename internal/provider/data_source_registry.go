package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RegistryDataSource{}

type RegistryDataSource struct {
	client *sdkclient.Client
}

func NewRegistryDataSource() datasource.DataSource {
	return &RegistryDataSource{}
}

func (d *RegistryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry"
}

func (d *RegistryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane container registry",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Registry ID",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "Registry URL",
			},
			"username": schema.StringAttribute{
				Computed:    true,
				Description: "Registry username",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Registry description",
			},
			"insecure": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the registry uses insecure connections",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the registry is enabled",
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

func (d *RegistryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type registryDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	URL         types.String `tfsdk:"url"`
	Username    types.String `tfsdk:"username"`
	Description types.String `tfsdk:"description"`
	Insecure    types.Bool   `tfsdk:"insecure"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (d *RegistryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config registryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	registry, err := d.client.GetContainerRegistry(ctx, config.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("registry not found", "No registry with id: "+config.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read registry", err.Error())
		return
	}

	state := registryDataSourceModel{
		ID:          types.StringValue(registry.ID),
		URL:         types.StringValue(registry.URL),
		Username:    types.StringValue(registry.Username),
		Description: types.StringValue(registry.Description),
		Insecure:    types.BoolValue(registry.Insecure),
		Enabled:     types.BoolValue(registry.Enabled),
		CreatedAt:   types.StringValue(registry.CreatedAt),
		UpdatedAt:   types.StringValue(registry.UpdatedAt),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
