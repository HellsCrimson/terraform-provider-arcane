package provider

import (
	"context"
	"sort"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &GlobalVariableResource{}
var _ resource.ResourceWithImportState = &GlobalVariableResource{}
var _ resource.ResourceWithValidateConfig = &GlobalVariableResource{}

type GlobalVariableResource struct{ client *sdkclient.Client }

func NewGlobalVariableResource() resource.Resource { return &GlobalVariableResource{} }

func (r *GlobalVariableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_variable"
}

func (r *GlobalVariableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "An Arcane global variable, available to projects and templates of every environment it is scoped to.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"key": resourceschema.StringAttribute{
				Required:    true,
				Description: "Variable name.",
			},
			"value": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Value of a plain variable, stored in state. For a secret use `value_wo` instead.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("value_wo")),
				},
			},
			"value_wo":         writeOnlyAttribute("value", "Value of a secret variable (`is_secret = true`)."),
			"value_wo_version": writeOnlyVersionAttribute("value"),
			"is_secret": resourceschema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Hide the value in Arcane. Arcane never returns a secret's value. Defaults to true when `value_wo` is used, false otherwise.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"all_environments": resourceschema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Make the variable available in every environment. Defaults to true unless `environment_ids` is set.",
				PlanModifiers: []planmodifier.Bool{allEnvironmentsDefault{}},
			},
			"environment_ids": resourceschema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Environments the variable is available in, when `all_environments` is false.",
			},
			"inject_into_containers": resourceschema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Also inject the variable into the containers of those environments. Defaults to false.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"created_at": resourceschema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *GlobalVariableResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type globalVariableModel struct {
	ID                   types.String `tfsdk:"id"`
	Key                  types.String `tfsdk:"key"`
	Value                types.String `tfsdk:"value"`
	ValueWO              types.String `tfsdk:"value_wo"`
	ValueWOVersion       types.Int64  `tfsdk:"value_wo_version"`
	IsSecret             types.Bool   `tfsdk:"is_secret"`
	AllEnvironments      types.Bool   `tfsdk:"all_environments"`
	EnvironmentIDs       types.Set    `tfsdk:"environment_ids"`
	InjectIntoContainers types.Bool   `tfsdk:"inject_into_containers"`
	CreatedAt            types.String `tfsdk:"created_at"`
}

func (r *GlobalVariableResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg globalVariableModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !cfg.AllEnvironments.IsUnknown() && !cfg.AllEnvironments.IsNull() && !cfg.AllEnvironments.ValueBool() && cfg.EnvironmentIDs.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("environment_ids"), "missing environment_ids",
			"A variable not available in all environments needs at least one environment in environment_ids.")
	}
	if !cfg.IsSecret.IsUnknown() && !cfg.IsSecret.IsNull() && cfg.IsSecret.ValueBool() && !cfg.Value.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("value"), "secret value stored in state",
			"Use value_wo and value_wo_version for a secret variable, so the value is never stored in state.")
	}
}

// request builds the create/update body from the plan. sendValue controls
// whether the value is part of it; leaving it out keeps the stored one.
func (m globalVariableModel) request(ctx context.Context, config globalVariableModel, sendValue bool) (sdkclient.GlobalVariableRequest, []string) {
	key := m.Key.ValueString()
	body := sdkclient.GlobalVariableRequest{Key: &key}
	if sendValue {
		v := m.Value.ValueString()
		if !config.ValueWO.IsNull() && !config.ValueWO.IsUnknown() {
			v = config.ValueWO.ValueString()
		}
		body.Value = &v
	}
	secret := !config.ValueWO.IsNull()
	if !m.IsSecret.IsNull() && !m.IsSecret.IsUnknown() {
		secret = m.IsSecret.ValueBool()
	}
	body.IsSecret = &secret

	ids := []string{}
	if !m.EnvironmentIDs.IsNull() && !m.EnvironmentIDs.IsUnknown() {
		_ = m.EnvironmentIDs.ElementsAs(ctx, &ids, false)
	}
	all := len(ids) == 0
	if !m.AllEnvironments.IsNull() && !m.AllEnvironments.IsUnknown() {
		all = m.AllEnvironments.ValueBool()
	}
	body.AllEnvironments = &all
	body.EnvironmentIDs = &ids
	inject := boolValue(m.InjectIntoContainers)
	body.InjectIntoContainers = &inject
	return body, ids
}

func (r *GlobalVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config globalVariableModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, _ := plan.request(ctx, config, true)
	v, err := r.client.CreateGlobalVariable(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("create global variable failed", err.Error())
		return
	}
	state := plan
	state.ValueWO = types.StringNull()
	applyGlobalVariable(&state, v, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GlobalVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state globalVariableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, err := r.client.GetGlobalVariable(ctx, state.ID.ValueString())
	if err != nil {
		if r.client.IsResourceGone(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read global variable failed", err.Error())
		return
	}
	importing := state.Key.IsNull()
	applyGlobalVariable(&state, v, state)
	// A plain value is reported back, so drift shows (and an import gets it);
	// a secret's is not.
	if !v.IsSecret && (!state.Value.IsNull() || importing) {
		state.Value = types.StringValue(v.Value)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GlobalVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config, state globalVariableModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The write-only value is only re-sent when its version changes.
	sendValue := !plan.Value.Equal(state.Value) || !plan.ValueWOVersion.Equal(state.ValueWOVersion)
	body, _ := plan.request(ctx, config, sendValue)
	v, err := r.client.UpdateGlobalVariable(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("update global variable failed", err.Error())
		return
	}
	newState := plan
	newState.ID = state.ID
	newState.CreatedAt = state.CreatedAt
	newState.ValueWO = types.StringNull()
	applyGlobalVariable(&newState, v, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *GlobalVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state globalVariableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteGlobalVariable(ctx, state.ID.ValueString()); err != nil && !r.client.IsResourceGone(err) {
		resp.Diagnostics.AddError("delete global variable failed", err.Error())
	}
}

func (r *GlobalVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyGlobalVariable copies the server-owned fields into state. desired
// carries the configuration: an unset environment_ids stays null.
func applyGlobalVariable(state *globalVariableModel, v *sdkclient.TemplateVariable, desired globalVariableModel) {
	if v.ID != "" {
		state.ID = types.StringValue(v.ID)
	}
	state.Key = types.StringValue(v.Key)
	state.IsSecret = types.BoolValue(v.IsSecret)
	state.AllEnvironments = types.BoolValue(v.AllEnvironments)
	state.InjectIntoContainers = types.BoolValue(v.InjectIntoContainers)
	if state.CreatedAt.IsNull() || state.CreatedAt.IsUnknown() {
		state.CreatedAt = types.StringValue(v.CreatedAt)
	}
	if desired.EnvironmentIDs.IsNull() && len(v.EnvironmentIDs) == 0 {
		state.EnvironmentIDs = types.SetNull(types.StringType)
		return
	}
	ids := append([]string(nil), v.EnvironmentIDs...)
	sort.Strings(ids)
	elems := make([]attr.Value, 0, len(ids))
	for _, id := range ids {
		elems = append(elems, types.StringValue(id))
	}
	state.EnvironmentIDs = types.SetValueMust(types.StringType, elems)
}

// allEnvironmentsDefault plans an unset all_environments from environment_ids:
// true without a list of environments, false with one.
type allEnvironmentsDefault struct{}

func (allEnvironmentsDefault) Description(context.Context) string {
	return "Defaults to true unless environment_ids is set."
}

func (m allEnvironmentsDefault) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (allEnvironmentsDefault) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	var ids types.Set
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("environment_ids"), &ids)...)
	if ids.IsUnknown() {
		return
	}
	resp.PlanValue = types.BoolValue(ids.IsNull() || len(ids.Elements()) == 0)
}
