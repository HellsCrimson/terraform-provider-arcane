package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TemplateRegistryDataSource{}

type TemplateRegistryDataSource struct {
	client *sdkclient.Client
}

func NewTemplateRegistryDataSource() datasource.DataSource {
	return &TemplateRegistryDataSource{}
}

func (d *TemplateRegistryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template_registry"
}

func (d *TemplateRegistryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane template registry",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Template registry ID",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Registry name",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "Registry URL",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Registry description",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the registry is enabled",
			},
		},
	}
}

func (d *TemplateRegistryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type templateRegistryDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	URL         types.String `tfsdk:"url"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
}

func (d *TemplateRegistryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config templateRegistryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	registry, err := d.client.GetTemplateRegistry(ctx, config.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("template registry not found", "No template registry with id: "+config.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read template registry", err.Error())
		return
	}

	state := templateRegistryDataSourceModel{
		ID:          types.StringValue(registry.ID),
		Name:        types.StringValue(registry.Name),
		URL:         types.StringValue(registry.URL),
		Description: types.StringValue(registry.Description),
		Enabled:     types.BoolValue(registry.Enabled),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
