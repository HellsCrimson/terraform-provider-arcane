package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TemplateDataSource{}

type TemplateDataSource struct {
	client *sdkclient.Client
}

func NewTemplateDataSource() datasource.DataSource {
	return &TemplateDataSource{}
}

func (d *TemplateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template"
}

func (d *TemplateDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane template",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Template ID",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Template name",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Template description",
			},
			"content": schema.StringAttribute{
				Computed:    true,
				Description: "Docker Compose YAML content",
			},
			"env_content": schema.StringAttribute{
				Computed:    true,
				Description: "Environment variables template content",
			},
			"is_custom": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is a custom template",
			},
			"is_remote": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this template is from a remote registry",
			},
			"registry_id": schema.StringAttribute{
				Computed:    true,
				Description: "Registry ID if remote",
			},
		},
	}
}

func (d *TemplateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type templateDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Content     types.String `tfsdk:"content"`
	EnvContent  types.String `tfsdk:"env_content"`
	IsCustom    types.Bool   `tfsdk:"is_custom"`
	IsRemote    types.Bool   `tfsdk:"is_remote"`
	RegistryID  types.String `tfsdk:"registry_id"`
}

func (d *TemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config templateDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	template, err := d.client.GetTemplate(ctx, config.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("template not found", "No template with id: "+config.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read template", err.Error())
		return
	}

	state := templateDataSourceModel{
		ID:          types.StringValue(template.ID),
		Name:        types.StringValue(template.Name),
		Description: types.StringValue(template.Description),
		Content:     types.StringValue(template.Content),
		IsCustom:    types.BoolValue(template.IsCustom),
		IsRemote:    types.BoolValue(template.IsRemote),
	}

	if template.EnvContent != nil && *template.EnvContent != "" {
		state.EnvContent = types.StringValue(*template.EnvContent)
	} else {
		state.EnvContent = types.StringNull()
	}

	if template.RegistryID != nil && *template.RegistryID != "" {
		state.RegistryID = types.StringValue(*template.RegistryID)
	} else {
		state.RegistryID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
