package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ApiKeyResource{}
var _ resource.ResourceWithImportState = &ApiKeyResource{}

type ApiKeyResource struct {
	client *sdkclient.Client
}

func NewApiKeyResource() resource.Resource {
	return &ApiKeyResource{}
}

func (r *ApiKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *ApiKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manages an API key for programmatic access to Arcane.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the API key",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": resourceschema.StringAttribute{
				Required:    true,
				Description: "Name of the API key (1-255 characters)",
			},
			"description": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Optional description of the API key (max 1000 characters)",
			},
			"expires_at": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Optional expiration date for the API key (RFC3339 format, e.g., '2025-12-31T23:59:59Z')",
			},
			"key": resourceschema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The full API key secret. Only available on creation - cannot be retrieved later.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key_prefix": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Prefix of the API key for identification",
			},
			"user_id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "ID of the user who owns the API key",
			},
			"last_used_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Last time the API key was used",
			},
			"created_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
			},
		},
	}
}

func (r *ApiKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type apiKeyModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
	Key         types.String `tfsdk:"key"`
	KeyPrefix   types.String `tfsdk:"key_prefix"`
	UserID      types.String `tfsdk:"user_id"`
	LastUsedAt  types.String `tfsdk:"last_used_at"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *ApiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.CreateApiKeyRequest{
		Name: plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		body.Description = &v
	}
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		v := plan.ExpiresAt.ValueString()
		body.ExpiresAt = &v
	}

	apiKey, err := r.client.CreateApiKey(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("create api key failed", err.Error())
		return
	}

	state := apiKeyModel{
		ID:        types.StringValue(apiKey.ID),
		Name:      types.StringValue(apiKey.Name),
		Key:       types.StringValue(apiKey.Key), // Only available on creation
		KeyPrefix: types.StringValue(apiKey.KeyPrefix),
		UserID:    types.StringValue(apiKey.UserID),
		CreatedAt: types.StringValue(apiKey.CreatedAt),
	}

	if apiKey.Description != nil {
		state.Description = types.StringValue(*apiKey.Description)
	} else {
		state.Description = types.StringNull()
	}
	if apiKey.ExpiresAt != nil {
		state.ExpiresAt = types.StringValue(*apiKey.ExpiresAt)
	} else {
		state.ExpiresAt = types.StringNull()
	}
	if apiKey.UpdatedAt != nil {
		state.UpdatedAt = types.StringValue(*apiKey.UpdatedAt)
	} else {
		state.UpdatedAt = types.StringNull()
	}
	// LastUsedAt not available on creation
	state.LastUsedAt = types.StringNull()

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey, err := r.client.GetApiKey(ctx, state.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read api key failed", err.Error())
		return
	}

	state.Name = types.StringValue(apiKey.Name)
	state.KeyPrefix = types.StringValue(apiKey.KeyPrefix)
	state.UserID = types.StringValue(apiKey.UserID)
	state.CreatedAt = types.StringValue(apiKey.CreatedAt)

	if apiKey.Description != nil {
		state.Description = types.StringValue(*apiKey.Description)
	} else {
		state.Description = types.StringNull()
	}
	if apiKey.ExpiresAt != nil {
		state.ExpiresAt = types.StringValue(*apiKey.ExpiresAt)
	} else {
		state.ExpiresAt = types.StringNull()
	}
	if apiKey.UpdatedAt != nil {
		state.UpdatedAt = types.StringValue(*apiKey.UpdatedAt)
	} else {
		state.UpdatedAt = types.StringNull()
	}
	if apiKey.LastUsedAt != nil {
		state.LastUsedAt = types.StringValue(*apiKey.LastUsedAt)
	} else {
		state.LastUsedAt = types.StringNull()
	}
	// Key is not returned on read, preserve from state
	// state.Key remains unchanged

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan apiKeyModel
	var state apiKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.UpdateApiKeyRequest{}

	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		v := plan.Name.ValueString()
		body.Name = &v
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		body.Description = &v
	}
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		v := plan.ExpiresAt.ValueString()
		body.ExpiresAt = &v
	}

	apiKey, err := r.client.UpdateApiKey(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("update api key failed", err.Error())
		return
	}

	state = plan
	state.ID = types.StringValue(apiKey.ID)
	state.KeyPrefix = types.StringValue(apiKey.KeyPrefix)
	state.UserID = types.StringValue(apiKey.UserID)
	state.CreatedAt = types.StringValue(apiKey.CreatedAt)
	// Preserve the key from previous state (not returned on update)

	if apiKey.Description != nil {
		state.Description = types.StringValue(*apiKey.Description)
	} else {
		state.Description = types.StringNull()
	}
	if apiKey.ExpiresAt != nil {
		state.ExpiresAt = types.StringValue(*apiKey.ExpiresAt)
	} else {
		state.ExpiresAt = types.StringNull()
	}
	if apiKey.UpdatedAt != nil {
		state.UpdatedAt = types.StringValue(*apiKey.UpdatedAt)
	} else {
		state.UpdatedAt = types.StringNull()
	}
	if apiKey.LastUsedAt != nil {
		state.LastUsedAt = types.StringValue(*apiKey.LastUsedAt)
	} else {
		state.LastUsedAt = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteApiKey(ctx, state.ID.ValueString()); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete api key failed", err.Error())
	}
}

func (r *ApiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
