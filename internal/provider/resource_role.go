package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &RoleResource{}
var _ resource.ResourceWithImportState = &RoleResource{}

type RoleResource struct {
	client *sdkclient.Client
}

func NewRoleResource() resource.Resource { return &RoleResource{} }

func (r *RoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *RoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manages a custom RBAC role. Built-in roles (Admin/Editor/Deployer/Viewer) are " +
			"read-only and cannot be managed by this resource.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the role.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": resourceschema.StringAttribute{
				Required:    true,
				Description: "Display name of the role (1-100 characters).",
			},
			"description": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Optional human description (max 500 characters).",
			},
			"permissions": resourceschema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Permission strings granted by this role (at least one), e.g. 'containers:start'.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"built_in": resourceschema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is a built-in role (always false for managed roles).",
			},
			"assigned_user_count": resourceschema.Int64Attribute{
				Computed:    true,
				Description: "How many users currently hold an assignment to this role.",
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

func (r *RoleResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type roleModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	Permissions       types.Set    `tfsdk:"permissions"`
	BuiltIn           types.Bool   `tfsdk:"built_in"`
	AssignedUserCount types.Int64  `tfsdk:"assigned_user_count"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func roleStringSliceToSet(ctx context.Context, ss []string) (types.Set, diag.Diagnostics) {
	if len(ss) == 0 {
		return types.SetNull(types.StringType), nil
	}
	return types.SetValueFrom(ctx, types.StringType, ss)
}

func roleSetToStringSlice(ctx context.Context, s types.Set) ([]string, diag.Diagnostics) {
	var out []string
	if s.IsNull() || s.IsUnknown() {
		return out, nil
	}
	diags := s.ElementsAs(ctx, &out, false)
	return out, diags
}

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	perms, diags := roleSetToStringSlice(ctx, plan.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := sdkclient.RoleCreateRequest{
		Name:        plan.Name.ValueString(),
		Permissions: perms,
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		body.Description = &v
	}

	role, err := r.client.CreateRole(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("create role failed", err.Error())
		return
	}

	state, diags := buildRoleState(ctx, role, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetRole(ctx, state.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read role failed", err.Error())
		return
	}

	state.Name = types.StringValue(role.Name)
	if !state.Description.IsNull() || role.Description != "" {
		state.Description = stringOrNullEmpty(role.Description)
	}
	permSet, diags := roleStringSliceToSet(ctx, role.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Permissions = permSet
	state.BuiltIn = types.BoolValue(role.BuiltIn)
	state.AssignedUserCount = types.Int64Value(role.AssignedUserCount)
	// Leave created_at and updated_at unchanged to avoid plan inconsistency

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleModel
	var state roleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	perms, diags := roleSetToStringSlice(ctx, plan.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := sdkclient.RoleUpdateRequest{
		Name:        plan.Name.ValueString(),
		Permissions: perms,
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		body.Description = &v
	}

	role, err := r.client.UpdateRole(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("update role failed", err.Error())
		return
	}

	newState, diags := buildRoleState(ctx, role, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteRole(ctx, state.ID.ValueString()); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete role failed", err.Error())
	}
}

func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// buildRoleState builds the resource state from the API result, preserving the
// configured description so create/update stay plan-consistent.
func buildRoleState(ctx context.Context, role *sdkclient.Role, plan roleModel) (roleModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	permSet, d := roleStringSliceToSet(ctx, role.Permissions)
	diags.Append(d...)
	state := roleModel{
		ID:                types.StringValue(role.ID),
		Name:              types.StringValue(role.Name),
		Permissions:       permSet,
		BuiltIn:           types.BoolValue(role.BuiltIn),
		AssignedUserCount: types.Int64Value(role.AssignedUserCount),
		CreatedAt:         stringOrNullEmpty(role.CreatedAt),
		UpdatedAt:         stringOrNullEmpty(role.UpdatedAt),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		state.Description = plan.Description
	} else {
		state.Description = stringOrNullEmpty(role.Description)
	}
	return state, diags
}

// stringOrNullEmpty returns a null String for empty values, else the value.
func stringOrNullEmpty(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}
