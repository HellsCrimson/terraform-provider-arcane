package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
			"config":         resourceschema.MapAttribute{Optional: true, ElementType: types.StringType, Description: "Provider-specific config as string map"},
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
	ID            types.String `tfsdk:"id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	ProviderName  types.String `tfsdk:"provider_name"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Config        types.Map    `tfsdk:"config"`
}

func (r *NotificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan notificationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := plan.EnvironmentID.ValueString()

	body := sdkclient.NotificationUpdate{
		Provider: plan.ProviderName.ValueString(),
		Enabled:  plan.Enabled.ValueBool(),
		Config:   mapStringMapToAny(ctx, plan.Config),
	}
	out, err := r.client.UpsertNotification(ctx, envID, body)
	if err != nil {
		resp.Diagnostics.AddError("upsert notification failed", err.Error())
		return
	}
	state := notificationModel{
		ID:            types.StringValue(envID + ":" + out.Provider),
		EnvironmentID: plan.EnvironmentID,
		ProviderName:  types.StringValue(out.Provider),
		Enabled:       types.BoolValue(out.Enabled),
		Config:        anyMapToStringMap(ctx, out.Config),
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
	state.Config = anyMapToStringMap(ctx, out.Config)
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
	body := sdkclient.NotificationUpdate{
		Provider: state.ProviderName.ValueString(),
		Enabled:  plan.Enabled.ValueBool(),
		Config:   mapStringMapToAny(ctx, plan.Config),
	}
	out, err := r.client.UpsertNotification(ctx, envID, body)
	if err != nil {
		resp.Diagnostics.AddError("upsert notification failed", err.Error())
		return
	}
	state.Enabled = types.BoolValue(out.Enabled)
	state.Config = anyMapToStringMap(ctx, out.Config)
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

// map helpers (string<->any)
func mapStringMapToAny(ctx context.Context, m types.Map) map[string]any {
	out := map[string]any{}
	if m.IsNull() || m.IsUnknown() {
		return out
	}
	var tmp map[string]string
	_ = m.ElementsAs(ctx, &tmp, false)
	for k, v := range tmp {
		out[k] = v
	}
	return out
}

func anyMapToStringMap(ctx context.Context, m map[string]any) types.Map {
	if len(m) == 0 {
		return types.MapNull(types.StringType)
	}
	elems := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elems[k] = types.StringValue(toString(v))
	}
	mv, _ := types.MapValue(types.StringType, elems)
	return mv
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}
