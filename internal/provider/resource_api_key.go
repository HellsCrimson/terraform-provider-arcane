package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
			"permissions": resourceschema.SetNestedAttribute{
				Required: true,
				Description: "Permission grants for the key (at least one). Each grant is a permission string " +
					"with an optional environment scope (omit environment_id for a global grant). Cannot " +
					"exceed the creating user's own permissions.",
				NestedObject: resourceschema.NestedAttributeObject{
					Attributes: map[string]resourceschema.Attribute{
						"permission": resourceschema.StringAttribute{
							Required:    true,
							Description: "Permission string, e.g. 'containers:list'.",
						},
						"environment_id": resourceschema.StringAttribute{
							Optional:    true,
							Description: "Environment ID to scope the grant to; omit for a global grant.",
						},
					},
				},
			},
			"is_bootstrap": resourceschema.BoolAttribute{
				Computed:    true,
				Description: "Whether the API key is an auto-generated environment bootstrap key.",
			},
			"is_static": resourceschema.BoolAttribute{
				Computed:    true,
				Description: "Whether the API key is environment-managed and protected from deletion.",
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
	Permissions types.Set    `tfsdk:"permissions"`
	IsBootstrap types.Bool   `tfsdk:"is_bootstrap"`
	IsStatic    types.Bool   `tfsdk:"is_static"`
	Key         types.String `tfsdk:"key"`
	KeyPrefix   types.String `tfsdk:"key_prefix"`
	UserID      types.String `tfsdk:"user_id"`
	LastUsedAt  types.String `tfsdk:"last_used_at"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

type apiKeyPermissionModel struct {
	Permission    types.String `tfsdk:"permission"`
	EnvironmentID types.String `tfsdk:"environment_id"`
}

var apiKeyPermissionObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"permission":     types.StringType,
	"environment_id": types.StringType,
}}

// apiKeyPermissionsToGrants converts the configured permissions set into the
// API request payload.
func apiKeyPermissionsToGrants(ctx context.Context, set types.Set) ([]sdkclient.ApiKeyPermissionGrant, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}
	var models []apiKeyPermissionModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]sdkclient.ApiKeyPermissionGrant, 0, len(models))
	for _, m := range models {
		g := sdkclient.ApiKeyPermissionGrant{Permission: m.Permission.ValueString()}
		if !m.EnvironmentID.IsNull() && !m.EnvironmentID.IsUnknown() && m.EnvironmentID.ValueString() != "" {
			v := m.EnvironmentID.ValueString()
			g.EnvironmentID = &v
		}
		out = append(out, g)
	}
	return out, diags
}

// apiKeyPermissionsToSet converts API permission grants into a Terraform set.
func apiKeyPermissionsToSet(ctx context.Context, grants []sdkclient.ApiKeyPermissionGrant) (types.Set, diag.Diagnostics) {
	models := make([]apiKeyPermissionModel, 0, len(grants))
	for _, g := range grants {
		m := apiKeyPermissionModel{Permission: types.StringValue(g.Permission)}
		if g.EnvironmentID != nil && *g.EnvironmentID != "" {
			m.EnvironmentID = types.StringValue(*g.EnvironmentID)
		} else {
			m.EnvironmentID = types.StringNull()
		}
		models = append(models, m)
	}
	if len(models) == 0 {
		return types.SetNull(apiKeyPermissionObjectType), nil
	}
	return types.SetValueFrom(ctx, apiKeyPermissionObjectType, models)
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
	grants, diags := apiKeyPermissionsToGrants(ctx, plan.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	body.Permissions = grants

	apiKey, err := r.client.CreateApiKey(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("create api key failed", err.Error())
		return
	}

	state := apiKeyModel{
		ID:          types.StringValue(apiKey.ID),
		Name:        types.StringValue(apiKey.Name),
		Description: plan.Description,
		ExpiresAt:   plan.ExpiresAt,
		Permissions: plan.Permissions,
		IsBootstrap: types.BoolValue(apiKey.IsBootstrap),
		IsStatic:    types.BoolValue(apiKey.IsStatic),
		Key:         types.StringValue(apiKey.Key), // Only available on creation
		KeyPrefix:   types.StringValue(apiKey.KeyPrefix),
		UserID:      types.StringValue(apiKey.UserID),
		CreatedAt:   types.StringValue(apiKey.CreatedAt),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() && apiKey.Description != nil {
		state.Description = types.StringValue(*apiKey.Description)
	}
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() && apiKey.ExpiresAt != nil {
		state.ExpiresAt = types.StringValue(*apiKey.ExpiresAt)
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
	state.IsBootstrap = types.BoolValue(apiKey.IsBootstrap)
	state.IsStatic = types.BoolValue(apiKey.IsStatic)

	if !state.Permissions.IsNull() && !state.Permissions.IsUnknown() {
		permSet, diags := apiKeyPermissionsToSet(ctx, apiKey.Permissions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Permissions = permSet
	}

	if !state.Description.IsNull() && !state.Description.IsUnknown() && apiKey.Description != nil {
		state.Description = types.StringValue(*apiKey.Description)
	}
	if !state.ExpiresAt.IsNull() && !state.ExpiresAt.IsUnknown() && apiKey.ExpiresAt != nil {
		state.ExpiresAt = types.StringValue(*apiKey.ExpiresAt)
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
	grants, diags := apiKeyPermissionsToGrants(ctx, plan.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	body.Permissions = grants

	apiKey, err := r.client.UpdateApiKey(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("update api key failed", err.Error())
		return
	}

	state = plan
	// id and created_at keep their planned value. Both carry UseStateForUnknown,
	// so the plan promises the prior one, and an update that rewrote them from
	// the response would fail as an inconsistent result.
	state.KeyPrefix = types.StringValue(apiKey.KeyPrefix)
	state.UserID = types.StringValue(apiKey.UserID)
	state.IsBootstrap = types.BoolValue(apiKey.IsBootstrap)
	state.IsStatic = types.BoolValue(apiKey.IsStatic)
	// Preserve the key from previous state (not returned on update)

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() && apiKey.Description != nil {
		state.Description = types.StringValue(*apiKey.Description)
	} else {
		state.Description = plan.Description
	}
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() && apiKey.ExpiresAt != nil {
		state.ExpiresAt = types.StringValue(*apiKey.ExpiresAt)
	} else {
		state.ExpiresAt = plan.ExpiresAt
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
