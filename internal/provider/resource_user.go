package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
			"roles": resourceschema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Roles assigned to the user.",
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
	ID          types.String `tfsdk:"id"`
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
	DisplayName types.String `tfsdk:"display_name"`
	Email       types.String `tfsdk:"email"`
	Locale      types.String `tfsdk:"locale"`
	Roles       types.Set    `tfsdk:"roles"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles := setToStringSlice(ctx, plan.Roles)
	body := sdkclient.CreateUserRequest{
		Username: plan.Username.ValueString(),
		Password: plan.Password.ValueString(),
		Roles:    roles,
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
	if u.Display != nil {
		state.DisplayName = types.StringValue(*u.Display)
	} else {
		state.DisplayName = types.StringNull()
	}
	if u.Email != nil {
		state.Email = types.StringValue(*u.Email)
	} else {
		state.Email = types.StringNull()
	}
	if u.Locale != nil {
		state.Locale = types.StringValue(*u.Locale)
	} else {
		state.Locale = types.StringNull()
	}
	state.Roles = stringSliceToSet(ctx, u.Roles)

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
	if u.Display != nil {
		state.DisplayName = types.StringValue(*u.Display)
	} else {
		state.DisplayName = types.StringNull()
	}
	if u.Email != nil {
		state.Email = types.StringValue(*u.Email)
	} else {
		state.Email = types.StringNull()
	}
	if u.Locale != nil {
		state.Locale = types.StringValue(*u.Locale)
	} else {
		state.Locale = types.StringNull()
	}
	state.Roles = stringSliceToSet(ctx, u.Roles)

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
	body.Roles = setToStringSlice(ctx, plan.Roles)

	tflog.Info(ctx, "Updating Arcane user", map[string]any{"id": id})
	u, err := r.client.UpdateUser(ctx, id, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating user", err.Error())
		return
	}

	state.Username = types.StringValue(u.Username)
	state.CreatedAt = stringOrNull(u.CreatedAt)
	state.UpdatedAt = stringOrNull(u.UpdatedAt)
	if u.Display != nil {
		state.DisplayName = types.StringValue(*u.Display)
	} else {
		state.DisplayName = types.StringNull()
	}
	if u.Email != nil {
		state.Email = types.StringValue(*u.Email)
	} else {
		state.Email = types.StringNull()
	}
	if u.Locale != nil {
		state.Locale = types.StringValue(*u.Locale)
	} else {
		state.Locale = types.StringNull()
	}
	state.Roles = stringSliceToSet(ctx, u.Roles)
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
func setToStringSlice(ctx context.Context, s types.Set) []string {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	var out []string
	_ = s.ElementsAs(ctx, &out, false)
	return out
}

func stringSliceToSet(ctx context.Context, ss []string) types.Set {
	if len(ss) == 0 {
		return types.SetNull(types.StringType)
	}
	elems := make([]attr.Value, 0, len(ss))
	for _, s := range ss {
		elems = append(elems, types.StringValue(s))
	}
	v, _ := types.SetValue(types.StringType, elems)
	return v
}

func stringOrNull(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}
