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

var _ resource.Resource = &OidcRoleMappingResource{}
var _ resource.ResourceWithImportState = &OidcRoleMappingResource{}

type OidcRoleMappingResource struct {
	client *sdkclient.Client
}

func NewOidcRoleMappingResource() resource.Resource { return &OidcRoleMappingResource{} }

func (r *OidcRoleMappingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_role_mapping"
}

func (r *OidcRoleMappingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Maps an OIDC group/claim value to a role. On each OIDC login, a user's group claim " +
			"is matched against claim_value and matching rows become role assignments. Manages only " +
			"manual mappings; mappings declared via the OIDC_ROLE_MAPPINGS env var are read-only.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the mapping.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"claim_value": resourceschema.StringAttribute{
				Required:    true,
				Description: "OIDC claim value to match (e.g. a group name).",
			},
			"role_id": resourceschema.StringAttribute{
				Required:    true,
				Description: "ID of the role to assign when the claim matches.",
			},
			"environment_id": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Environment ID to scope the assignment to; omit for a global assignment.",
			},
			"source": resourceschema.StringAttribute{
				Computed:    true,
				Description: "How the mapping was created (manual or env).",
			},
			"created_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp.",
			},
		},
	}
}

func (r *OidcRoleMappingResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type oidcRoleMappingModel struct {
	ID            types.String `tfsdk:"id"`
	ClaimValue    types.String `tfsdk:"claim_value"`
	RoleID        types.String `tfsdk:"role_id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	Source        types.String `tfsdk:"source"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *OidcRoleMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan oidcRoleMappingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.OidcRoleMappingCreateRequest{
		ClaimValue: plan.ClaimValue.ValueString(),
		RoleID:     plan.RoleID.ValueString(),
	}
	if !plan.EnvironmentID.IsNull() && !plan.EnvironmentID.IsUnknown() && plan.EnvironmentID.ValueString() != "" {
		v := plan.EnvironmentID.ValueString()
		body.EnvironmentID = &v
	}

	m, err := r.client.CreateOidcRoleMapping(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("create oidc role mapping failed", err.Error())
		return
	}

	state := buildOidcRoleMappingState(m, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OidcRoleMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state oidcRoleMappingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// There is no get-by-id endpoint; list and find the row.
	mappings, err := r.client.ListOidcRoleMappings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("read oidc role mapping failed", err.Error())
		return
	}
	id := state.ID.ValueString()
	var found *sdkclient.OidcRoleMapping
	for i := range mappings {
		if mappings[i].ID == id {
			found = &mappings[i]
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ClaimValue = types.StringValue(found.ClaimValue)
	state.RoleID = types.StringValue(found.RoleID)
	if !state.EnvironmentID.IsNull() || found.EnvironmentID != "" {
		state.EnvironmentID = stringOrNullEmpty(found.EnvironmentID)
	}
	state.Source = types.StringValue(found.Source)
	// Leave created_at and updated_at unchanged to avoid plan inconsistency
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OidcRoleMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan oidcRoleMappingModel
	var state oidcRoleMappingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.OidcRoleMappingUpdateRequest{
		ClaimValue: plan.ClaimValue.ValueString(),
		RoleID:     plan.RoleID.ValueString(),
	}
	if !plan.EnvironmentID.IsNull() && !plan.EnvironmentID.IsUnknown() && plan.EnvironmentID.ValueString() != "" {
		v := plan.EnvironmentID.ValueString()
		body.EnvironmentID = &v
	}

	m, err := r.client.UpdateOidcRoleMapping(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("update oidc role mapping failed", err.Error())
		return
	}

	newState := buildOidcRoleMappingState(m, plan)
	newState.CreatedAt = state.CreatedAt
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *OidcRoleMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state oidcRoleMappingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteOidcRoleMapping(ctx, state.ID.ValueString()); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete oidc role mapping failed", err.Error())
	}
}

func (r *OidcRoleMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildOidcRoleMappingState(m *sdkclient.OidcRoleMapping, plan oidcRoleMappingModel) oidcRoleMappingModel {
	state := oidcRoleMappingModel{
		ID:         types.StringValue(m.ID),
		ClaimValue: types.StringValue(m.ClaimValue),
		RoleID:     types.StringValue(m.RoleID),
		Source:     types.StringValue(m.Source),
		CreatedAt:  stringOrNullEmpty(m.CreatedAt),
		UpdatedAt:  stringOrNullEmpty(m.UpdatedAt),
	}
	if !plan.EnvironmentID.IsNull() && !plan.EnvironmentID.IsUnknown() {
		state.EnvironmentID = plan.EnvironmentID
	} else {
		state.EnvironmentID = stringOrNullEmpty(m.EnvironmentID)
	}
	return state
}
