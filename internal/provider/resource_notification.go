package provider

import (
	"context"
	"fmt"
	"math/big"
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

var _ resource.Resource = &NotificationResource{}
var _ resource.ResourceWithImportState = &NotificationResource{}

type NotificationResource struct{ client *sdkclient.Client }

func NewNotificationResource() resource.Resource { return &NotificationResource{} }

func (r *NotificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification"
}

func (r *NotificationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"id":             resourceschema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"environment_id": resourceschema.StringAttribute{Required: true, Description: "Environment ID"},
			"provider_name":  resourceschema.StringAttribute{Required: true, Description: "Notification provider name"},
			"enabled":        resourceschema.BoolAttribute{Required: true},
			"config":         resourceschema.DynamicAttribute{Optional: true, Description: "Provider-specific config object"},
		},
	}
}

func (r *NotificationResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type notificationModel struct {
	ID            types.String  `tfsdk:"id"`
	EnvironmentID types.String  `tfsdk:"environment_id"`
	ProviderName  types.String  `tfsdk:"provider_name"`
	Enabled       types.Bool    `tfsdk:"enabled"`
	Config        types.Dynamic `tfsdk:"config"`
}

func (r *NotificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan notificationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := plan.EnvironmentID.ValueString()
	config, diags := dynamicConfigToAnyMap(plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.NotificationUpdate{
		Provider: plan.ProviderName.ValueString(),
		Enabled:  plan.Enabled.ValueBool(),
		Config:   config,
	}
	out, err := r.client.UpsertNotification(ctx, envID, body)
	if err != nil {
		resp.Diagnostics.AddError("upsert notification failed", err.Error())
		return
	}
	state := notificationModel{
		ID:            types.StringValue(envID + ":" + out.Provider),
		EnvironmentID: plan.EnvironmentID,
		ProviderName:  plan.ProviderName,
		Enabled:       types.BoolValue(out.Enabled),
		Config:        plan.Config,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state notificationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := state.EnvironmentID.ValueString()
	provider := state.ProviderName.ValueString()
	out, err := r.client.GetNotification(ctx, envID, provider)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read notification failed", err.Error())
		return
	}
	state.Enabled = types.BoolValue(out.Enabled)
	state.ID = types.StringValue(envID + ":" + provider)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotificationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan notificationModel
	var state notificationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envID := state.EnvironmentID.ValueString()
	config, diags := dynamicConfigToAnyMap(plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.NotificationUpdate{
		Provider: state.ProviderName.ValueString(),
		Enabled:  plan.Enabled.ValueBool(),
		Config:   config,
	}
	out, err := r.client.UpsertNotification(ctx, envID, body)
	if err != nil {
		resp.Diagnostics.AddError("upsert notification failed", err.Error())
		return
	}
	state.Enabled = types.BoolValue(out.Enabled)
	state.Config = plan.Config
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotificationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state notificationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteNotification(ctx, state.EnvironmentID.ValueString(), state.ProviderName.ValueString()); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete notification failed", err.Error())
	}
}

func (r *NotificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// envID:provider
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("invalid import id", "expected env_id:provider")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("provider_name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// config helpers (dynamic Terraform value <-> JSON-compatible map)
func dynamicConfigToAnyMap(m types.Dynamic) (map[string]any, diag.Diagnostics) {
	out := map[string]any{}
	if m.IsNull() || m.IsUnknown() {
		return out, nil
	}
	root := m.UnderlyingValue()
	if root == nil || root.IsNull() || root.IsUnknown() {
		return out, nil
	}
	value := attrValueToAny(root)
	config, ok := value.(map[string]any)
	if !ok {
		return nil, diag.Diagnostics{
			diag.NewErrorDiagnostic("invalid notification config", "The config attribute must be an object."),
		}
	}
	return config, nil
}

func anyMapToDynamicConfig(ctx context.Context, m map[string]any) types.Dynamic {
	if len(m) == 0 {
		return types.DynamicNull()
	}
	value := anyToAttrValue(ctx, m)
	dynamic, ok := value.(types.Dynamic)
	if ok {
		return dynamic
	}
	return types.DynamicValue(value)
}

func attrValueToAny(v attr.Value) any {
	if v == nil || v.IsNull() || v.IsUnknown() {
		return nil
	}
	switch t := v.(type) {
	case types.Dynamic:
		return attrValueToAny(t.UnderlyingValue())
	case types.String:
		return t.ValueString()
	case types.Bool:
		return t.ValueBool()
	case types.Number:
		f, _ := t.ValueBigFloat().Float64()
		return f
	case types.Object:
		out := make(map[string]any, len(t.Attributes()))
		for k, v := range t.Attributes() {
			out[k] = attrValueToAny(v)
		}
		return out
	case types.Map:
		out := make(map[string]any, len(t.Elements()))
		for k, v := range t.Elements() {
			out[k] = attrValueToAny(v)
		}
		return out
	case types.List:
		out := make([]any, len(t.Elements()))
		for i, v := range t.Elements() {
			out[i] = attrValueToAny(v)
		}
		return out
	case types.Tuple:
		out := make([]any, len(t.Elements()))
		for i, v := range t.Elements() {
			out[i] = attrValueToAny(v)
		}
		return out
	case types.Set:
		out := make([]any, len(t.Elements()))
		for i, v := range t.Elements() {
			out[i] = attrValueToAny(v)
		}
		return out
	default:
		return v.String()
	}
}

func anyToAttrValue(ctx context.Context, v any) attr.Value {
	switch t := v.(type) {
	case nil:
		return types.DynamicNull()
	case string:
		return types.StringValue(t)
	case []byte:
		return types.StringValue(string(t))
	case bool:
		return types.BoolValue(t)
	case int:
		return types.NumberValue(big.NewFloat(float64(t)))
	case int64:
		return types.NumberValue(big.NewFloat(float64(t)))
	case float64:
		return types.NumberValue(big.NewFloat(t))
	case map[string]any:
		attrTypes := make(map[string]attr.Type, len(t))
		attrs := make(map[string]attr.Value, len(t))
		for k, v := range t {
			attrValue := anyToAttrValue(ctx, v)
			attrTypes[k] = attrValue.Type(ctx)
			attrs[k] = attrValue
		}
		value, diags := types.ObjectValue(attrTypes, attrs)
		if diags.HasError() {
			return types.DynamicNull()
		}
		return value
	case []any:
		elemTypes := make([]attr.Type, len(t))
		elems := make([]attr.Value, len(t))
		for i, v := range t {
			elem := anyToAttrValue(ctx, v)
			elemTypes[i] = elem.Type(ctx)
			elems[i] = elem
		}
		value, diags := types.TupleValue(elemTypes, elems)
		if diags.HasError() {
			return types.DynamicNull()
		}
		return value
	default:
		return types.StringValue(fmt.Sprint(t))
	}
}
