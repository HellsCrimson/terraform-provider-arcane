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

var _ resource.Resource = &NetworkResource{}
var _ resource.ResourceWithImportState = &NetworkResource{}

type NetworkResource struct {
	client *sdkclient.Client
}

func NewNetworkResource() resource.Resource {
	return &NetworkResource{}
}

func (r *NetworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (r *NetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manages a Docker network for container communication.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the network",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"environment_id": resourceschema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": resourceschema.StringAttribute{
				Required:    true,
				Description: "Name of the network",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"driver": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Network driver (e.g., bridge, overlay, host, macvlan). Defaults to 'bridge'.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"attachable": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Allow manual container attachment",
			},
			"internal": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Restrict external access to the network",
			},
			"enable_ipv6": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Enable IPv6 networking",
			},
			"check_duplicate": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Check for duplicate network names",
			},
			"ingress": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Enable routing-mesh for swarm cluster",
			},
			"labels": resourceschema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "User-defined labels for metadata",
			},
			"options": resourceschema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Driver-specific options",
			},
			// Computed fields
			"scope": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Scope of the network (local or swarm)",
			},
			"created": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
		},
	}
}

func (r *NetworkResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type networkModel struct {
	ID             types.String `tfsdk:"id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
	Name           types.String `tfsdk:"name"`
	Driver         types.String `tfsdk:"driver"`
	Attachable     types.Bool   `tfsdk:"attachable"`
	Internal       types.Bool   `tfsdk:"internal"`
	EnableIPv6     types.Bool   `tfsdk:"enable_ipv6"`
	CheckDuplicate types.Bool   `tfsdk:"check_duplicate"`
	Ingress        types.Bool   `tfsdk:"ingress"`
	Labels         types.Map    `tfsdk:"labels"`
	Options        types.Map    `tfsdk:"options"`
	Scope          types.String `tfsdk:"scope"`
	Created        types.String `tfsdk:"created"`
}

func (r *NetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan networkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := sdkclient.NetworkCreateOptions{}

	if !plan.Driver.IsNull() && !plan.Driver.IsUnknown() {
		v := plan.Driver.ValueString()
		opts.Driver = &v
	}
	if !plan.Attachable.IsNull() && !plan.Attachable.IsUnknown() {
		v := plan.Attachable.ValueBool()
		opts.Attachable = &v
	}
	if !plan.Internal.IsNull() && !plan.Internal.IsUnknown() {
		v := plan.Internal.ValueBool()
		opts.Internal = &v
	}
	if !plan.EnableIPv6.IsNull() && !plan.EnableIPv6.IsUnknown() {
		v := plan.EnableIPv6.ValueBool()
		opts.EnableIPv6 = &v
	}
	if !plan.CheckDuplicate.IsNull() && !plan.CheckDuplicate.IsUnknown() {
		v := plan.CheckDuplicate.ValueBool()
		opts.CheckDuplicate = &v
	}
	if !plan.Ingress.IsNull() && !plan.Ingress.IsUnknown() {
		v := plan.Ingress.ValueBool()
		opts.Ingress = &v
	}
	if !plan.Labels.IsNull() && !plan.Labels.IsUnknown() {
		opts.Labels = mapFromStringMap(ctx, plan.Labels)
	}
	if !plan.Options.IsNull() && !plan.Options.IsUnknown() {
		opts.Options = mapFromStringMap(ctx, plan.Options)
	}

	body := sdkclient.NetworkCreateRequest{
		Name:    plan.Name.ValueString(),
		Options: opts,
	}

	network, err := r.client.CreateNetwork(ctx, plan.EnvironmentID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("create network failed", err.Error())
		return
	}

	// Get full network details
	networkDetails, err := r.client.GetNetwork(ctx, plan.EnvironmentID.ValueString(), network.ID)
	if err != nil {
		resp.Diagnostics.AddError("read network after create failed", err.Error())
		return
	}

	state := networkModel{
		ID:            types.StringValue(networkDetails.ID),
		EnvironmentID: plan.EnvironmentID,
		Name:          types.StringValue(networkDetails.Name),
		Driver:        types.StringValue(networkDetails.Driver),
		Attachable:    types.BoolValue(networkDetails.Attachable),
		Internal:      types.BoolValue(networkDetails.Internal),
		EnableIPv6:    types.BoolValue(networkDetails.EnableIPv6),
		Scope:         types.StringValue(networkDetails.Scope),
		Created:       types.StringValue(networkDetails.Created),
		Labels:        plan.Labels,
		Options:       plan.Options,
	}

	// Preserve optional bools from plan
	state.CheckDuplicate = plan.CheckDuplicate
	state.Ingress = plan.Ingress

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state networkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	network, err := r.client.GetNetwork(ctx, state.EnvironmentID.ValueString(), state.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read network failed", err.Error())
		return
	}

	state.Name = types.StringValue(network.Name)
	state.Driver = types.StringValue(network.Driver)
	state.Attachable = types.BoolValue(network.Attachable)
	state.Internal = types.BoolValue(network.Internal)
	state.EnableIPv6 = types.BoolValue(network.EnableIPv6)
	state.Scope = types.StringValue(network.Scope)
	state.Created = types.StringValue(network.Created)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Networks cannot be updated - all changes require replacement
	resp.Diagnostics.AddError("update not supported", "Networks cannot be updated in place. All changes require replacement.")
}

func (r *NetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state networkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteNetwork(ctx, state.EnvironmentID.ValueString(), state.ID.ValueString()); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete network failed", err.Error())
	}
}

func (r *NetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: environment_id/network_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("invalid import ID", "Expected format: environment_id/network_id")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
