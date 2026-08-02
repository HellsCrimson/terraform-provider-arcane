package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &UserResource{}
var _ resource.ResourceWithImportState = &UserResource{}

type UserResource struct {
	client *sdkclient.Client
}

func NewUserResource() resource.Resource { return &UserResource{} }

func (r *UserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": resourceschema.StringAttribute{
				Required:    true,
				Description: "Username of the user. Changing forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": resourceschema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Password for the user. Stored in state as sensitive when using older Terraform/OpenTofu runtimes.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(8),
				},
			},
			"display_name": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Display name of the user.",
			},
			"email": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Email address of the user.",
			},
			"locale": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Locale preference (e.g., en-US).",
			},
			"role_assignments": resourceschema.SetNestedAttribute{
				Optional: true,
				Description: "Manual role assignments for the user. Each entry references a role by ID " +
					"and may optionally be scoped to an environment (omit environment_id for a global " +
					"assignment). This manages only manual assignments; assignments created via OIDC are " +
					"left untouched.",
				NestedObject: resourceschema.NestedAttributeObject{
					Attributes: map[string]resourceschema.Attribute{
						"role_id": resourceschema.StringAttribute{
							Required:    true,
							Description: "ID of the role to grant.",
						},
						"environment_id": resourceschema.StringAttribute{
							Optional:    true,
							Description: "Environment ID to scope the assignment to; omit for a global assignment.",
						},
					},
				},
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

func (r *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	if c, ok := req.ProviderData.(*sdkclient.Client); ok {
		r.client = c
	}
}

type userModel struct {
	ID              types.String `tfsdk:"id"`
	Username        types.String `tfsdk:"username"`
	Password        types.String `tfsdk:"password"`
	DisplayName     types.String `tfsdk:"display_name"`
	Email           types.String `tfsdk:"email"`
	Locale          types.String `tfsdk:"locale"`
	RoleAssignments types.Set    `tfsdk:"role_assignments"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

type roleAssignmentModel struct {
	RoleID        types.String `tfsdk:"role_id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
}

var roleAssignmentObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"role_id":        types.StringType,
	"environment_id": types.StringType,
}}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.CreateUserRequest{
		Username: plan.Username.ValueString(),
		Password: plan.Password.ValueString(),
	}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		v := plan.DisplayName.ValueString()
		body.DisplayName = &v
	}
	if !plan.Email.IsNull() && !plan.Email.IsUnknown() {
		v := plan.Email.ValueString()
		body.Email = &v
	}
	if !plan.Locale.IsNull() && !plan.Locale.IsUnknown() {
		v := plan.Locale.ValueString()
		body.Locale = &v
	}

	tflog.Info(ctx, "Creating Arcane user", map[string]any{"username": body.Username})
	u, err := r.client.CreateUser(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}

	state := userModel{
		ID:        types.StringValue(u.ID),
		Username:  types.StringValue(u.Username),
		CreatedAt: stringOrNull(u.CreatedAt),
		UpdatedAt: stringOrNull(u.UpdatedAt),
	}
	// Keep provided password in state to avoid sensitive inconsistency after apply
	state.Password = plan.Password
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() && u.Display != nil {
		state.DisplayName = types.StringValue(*u.Display)
	} else {
		state.DisplayName = plan.DisplayName
	}
	if !plan.Email.IsNull() && !plan.Email.IsUnknown() && u.Email != nil {
		state.Email = types.StringValue(*u.Email)
	} else {
		state.Email = plan.Email
	}
	if !plan.Locale.IsNull() && !plan.Locale.IsUnknown() && u.Locale != nil {
		state.Locale = types.StringValue(*u.Locale)
	} else {
		state.Locale = plan.Locale
	}
	// Apply manual role assignments via the dedicated endpoint (the create
	// endpoint does not accept them).
	if !plan.RoleAssignments.IsNull() && !plan.RoleAssignments.IsUnknown() {
		inputs, diags := roleAssignmentsToInputs(ctx, plan.RoleAssignments)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.SetUserRoleAssignments(ctx, u.ID, inputs); err != nil {
			resp.Diagnostics.AddError("Error setting user role assignments", err.Error())
			return
		}
		state.RoleAssignments = plan.RoleAssignments
	} else {
		state.RoleAssignments = types.SetNull(roleAssignmentObjectType)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	u, err := r.client.GetUser(ctx, id)
	if err != nil {
		// If the user is gone, drop from state
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	state.Username = types.StringValue(u.Username)
	state.CreatedAt = stringOrNull(u.CreatedAt)
	state.UpdatedAt = stringOrNull(u.UpdatedAt)
	if !state.DisplayName.IsNull() && !state.DisplayName.IsUnknown() && u.Display != nil {
		state.DisplayName = types.StringValue(*u.Display)
	}
	if !state.Email.IsNull() && !state.Email.IsUnknown() && u.Email != nil {
		state.Email = types.StringValue(*u.Email)
	}
	if !state.Locale.IsNull() && !state.Locale.IsUnknown() && u.Locale != nil {
		state.Locale = types.StringValue(*u.Locale)
	}
	if !state.RoleAssignments.IsNull() && !state.RoleAssignments.IsUnknown() {
		set, diags := roleAssignmentsToSet(ctx, u.RoleAssignments)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.RoleAssignments = set
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userModel
	var state userModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	body := sdkclient.UpdateUserRequest{}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		v := plan.DisplayName.ValueString()
		body.DisplayName = &v
	}
	if !plan.Email.IsNull() && !plan.Email.IsUnknown() {
		v := plan.Email.ValueString()
		body.Email = &v
	}
	if !plan.Locale.IsNull() && !plan.Locale.IsUnknown() {
		v := plan.Locale.ValueString()
		body.Locale = &v
	}
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() && plan.Password.ValueString() != "" {
		v := plan.Password.ValueString()
		body.Password = &v
	}

	tflog.Info(ctx, "Updating Arcane user", map[string]any{"id": id})
	u, err := r.client.UpdateUser(ctx, id, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating user", err.Error())
		return
	}

	// Reconcile manual role assignments when configured.
	if !plan.RoleAssignments.IsNull() && !plan.RoleAssignments.IsUnknown() {
		inputs, diags := roleAssignmentsToInputs(ctx, plan.RoleAssignments)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.SetUserRoleAssignments(ctx, id, inputs); err != nil {
			resp.Diagnostics.AddError("Error setting user role assignments", err.Error())
			return
		}
	}

	state.Username = types.StringValue(u.Username)
	// created_at keeps its planned value: UseStateForUnknown makes the plan
	// promise the prior one, so rewriting it here would fail the apply as an
	// inconsistent result.
	state.UpdatedAt = stringOrNull(u.UpdatedAt)
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() && u.Display != nil {
		state.DisplayName = types.StringValue(*u.Display)
	} else {
		state.DisplayName = plan.DisplayName
	}
	if !plan.Email.IsNull() && !plan.Email.IsUnknown() && u.Email != nil {
		state.Email = types.StringValue(*u.Email)
	} else {
		state.Email = plan.Email
	}
	if !plan.Locale.IsNull() && !plan.Locale.IsUnknown() && u.Locale != nil {
		state.Locale = types.StringValue(*u.Locale)
	} else {
		state.Locale = plan.Locale
	}
	if !plan.RoleAssignments.IsNull() && !plan.RoleAssignments.IsUnknown() {
		state.RoleAssignments = plan.RoleAssignments
	} else {
		state.RoleAssignments = types.SetNull(roleAssignmentObjectType)
	}
	// If password provided in plan, keep it in state to match planned value
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() && plan.Password.ValueString() != "" {
		state.Password = plan.Password
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	tflog.Info(ctx, "Deleting Arcane user", map[string]any{"id": id})
	if err := r.client.DeleteUser(ctx, id); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			// already gone
		} else {
			resp.Diagnostics.AddError("Error deleting user", err.Error())
			return
		}
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by ID
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func stringOrNull(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

// roleAssignmentsToInputs converts the configured role_assignments set into the
// API request payload.
func roleAssignmentsToInputs(ctx context.Context, set types.Set) ([]sdkclient.RoleAssignmentInput, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return []sdkclient.RoleAssignmentInput{}, diags
	}
	var models []roleAssignmentModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]sdkclient.RoleAssignmentInput, 0, len(models))
	for _, m := range models {
		in := sdkclient.RoleAssignmentInput{RoleID: m.RoleID.ValueString()}
		if !m.EnvironmentID.IsNull() && !m.EnvironmentID.IsUnknown() && m.EnvironmentID.ValueString() != "" {
			v := m.EnvironmentID.ValueString()
			in.EnvironmentID = &v
		}
		out = append(out, in)
	}
	return out, diags
}

// roleAssignmentsToSet converts the role assignments returned by the API into a
// Terraform set, keeping only manually-managed assignments (OIDC-sourced ones
// are managed outside Terraform).
func roleAssignmentsToSet(ctx context.Context, assignments []sdkclient.UserRoleAssignment) (types.Set, diag.Diagnostics) {
	manual := make([]roleAssignmentModel, 0, len(assignments))
	for _, a := range assignments {
		if a.Source != "" && a.Source != "manual" {
			continue
		}
		m := roleAssignmentModel{RoleID: types.StringValue(a.RoleID)}
		if a.EnvironmentID != "" {
			m.EnvironmentID = types.StringValue(a.EnvironmentID)
		} else {
			m.EnvironmentID = types.StringNull()
		}
		manual = append(manual, m)
	}
	if len(manual) == 0 {
		return types.SetNull(roleAssignmentObjectType), nil
	}
	return types.SetValueFrom(ctx, roleAssignmentObjectType, manual)
}
