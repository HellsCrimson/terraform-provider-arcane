package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ApiKeyDataSource{}

type ApiKeyDataSource struct {
	client *sdkclient.Client
}

func NewApiKeyDataSource() datasource.DataSource {
	return &ApiKeyDataSource{}
}

func (d *ApiKeyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (d *ApiKeyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane API key (note: full key secret is not retrievable after creation)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "API key ID",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the API key",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of the API key",
			},
			"expires_at": schema.StringAttribute{
				Computed:    true,
				Description: "Expiration date",
			},
			"key_prefix": schema.StringAttribute{
				Computed:    true,
				Description: "Key prefix for identification",
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "Owner user ID",
			},
			"last_used_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last usage timestamp",
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

func (d *ApiKeyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type apiKeyDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
	KeyPrefix   types.String `tfsdk:"key_prefix"`
	UserID      types.String `tfsdk:"user_id"`
	LastUsedAt  types.String `tfsdk:"last_used_at"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (d *ApiKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config apiKeyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey, err := d.client.GetApiKey(ctx, config.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("api key not found", "No API key with id: "+config.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read api key", err.Error())
		return
	}

	state := apiKeyDataSourceModel{
		ID:        types.StringValue(apiKey.ID),
		Name:      types.StringValue(apiKey.Name),
		KeyPrefix: types.StringValue(apiKey.KeyPrefix),
		UserID:    types.StringValue(apiKey.UserID),
		CreatedAt: types.StringValue(apiKey.CreatedAt),
	}

	if apiKey.Description != nil && *apiKey.Description != "" {
		state.Description = types.StringValue(*apiKey.Description)
	} else {
		state.Description = types.StringNull()
	}
	if apiKey.ExpiresAt != nil && *apiKey.ExpiresAt != "" {
		state.ExpiresAt = types.StringValue(*apiKey.ExpiresAt)
	} else {
		state.ExpiresAt = types.StringNull()
	}
	if apiKey.LastUsedAt != nil && *apiKey.LastUsedAt != "" {
		state.LastUsedAt = types.StringValue(*apiKey.LastUsedAt)
	} else {
		state.LastUsedAt = types.StringNull()
	}
	if apiKey.UpdatedAt != nil && *apiKey.UpdatedAt != "" {
		state.UpdatedAt = types.StringValue(*apiKey.UpdatedAt)
	} else {
		state.UpdatedAt = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
