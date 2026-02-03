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

var _ resource.Resource = &VolumeResource{}
var _ resource.ResourceWithImportState = &VolumeResource{}

type VolumeResource struct {
	client *sdkclient.Client
}

func NewVolumeResource() resource.Resource {
	return &VolumeResource{}
}

func (r *VolumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (r *VolumeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manages a Docker volume for persistent storage.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the volume",
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
				Description: "Name of the volume",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"driver": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Volume driver (e.g., local, nfs). Defaults to 'local'.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"driver_opts": resourceschema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Driver-specific options",
			},
			"labels": resourceschema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "User-defined labels for metadata",
			},
			// Computed fields
			"mountpoint": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Mount point of the volume on the host",
			},
			"scope": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Scope of the volume (local or global)",
			},
			"created_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
			"in_use": resourceschema.BoolAttribute{
				Computed:    true,
				Description: "Whether the volume is currently in use",
			},
			"size": resourceschema.Int64Attribute{
				Computed:    true,
				Description: "Size of the volume in bytes",
			},
			"containers": resourceschema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of containers using this volume",
			},
		},
	}
}

func (r *VolumeResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type volumeModel struct {
	ID            types.String `tfsdk:"id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	Name          types.String `tfsdk:"name"`
	Driver        types.String `tfsdk:"driver"`
	DriverOpts    types.Map    `tfsdk:"driver_opts"`
	Labels        types.Map    `tfsdk:"labels"`
	Mountpoint    types.String `tfsdk:"mountpoint"`
	Scope         types.String `tfsdk:"scope"`
	CreatedAt     types.String `tfsdk:"created_at"`
	InUse         types.Bool   `tfsdk:"in_use"`
	Size          types.Int64  `tfsdk:"size"`
	Containers    types.List   `tfsdk:"containers"`
}

func (r *VolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan volumeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.CreateVolumeRequest{
		Name: plan.Name.ValueString(),
	}

	if !plan.Driver.IsNull() && !plan.Driver.IsUnknown() {
		v := plan.Driver.ValueString()
		body.Driver = &v
	}
	if !plan.DriverOpts.IsNull() && !plan.DriverOpts.IsUnknown() {
		body.DriverOpts = mapFromStringMap(ctx, plan.DriverOpts)
	}
	if !plan.Labels.IsNull() && !plan.Labels.IsUnknown() {
		body.Labels = mapFromStringMap(ctx, plan.Labels)
	}

	volume, err := r.client.CreateVolume(ctx, plan.EnvironmentID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("create volume failed", err.Error())
		return
	}

	state := volumeModel{
		ID:            types.StringValue(volume.ID),
		EnvironmentID: plan.EnvironmentID,
		Name:          types.StringValue(volume.Name),
		Driver:        types.StringValue(volume.Driver),
		Mountpoint:    types.StringValue(volume.Mountpoint),
		Scope:         types.StringValue(volume.Scope),
		CreatedAt:     types.StringValue(volume.CreatedAt),
		InUse:         types.BoolValue(volume.InUse),
		Size:          types.Int64Value(volume.Size),
		DriverOpts:    plan.DriverOpts,
		Labels:        plan.Labels,
	}

	if len(volume.Containers) > 0 {
		state.Containers = stringsToList(ctx, volume.Containers)
	} else {
		state.Containers = types.ListNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state volumeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	volume, err := r.client.GetVolume(ctx, state.EnvironmentID.ValueString(), state.Name.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read volume failed", err.Error())
		return
	}

	state.ID = types.StringValue(volume.ID)
	state.Name = types.StringValue(volume.Name)
	state.Driver = types.StringValue(volume.Driver)
	state.Mountpoint = types.StringValue(volume.Mountpoint)
	state.Scope = types.StringValue(volume.Scope)
	state.CreatedAt = types.StringValue(volume.CreatedAt)
	state.InUse = types.BoolValue(volume.InUse)
	state.Size = types.Int64Value(volume.Size)

	if len(volume.Containers) > 0 {
		state.Containers = stringsToList(ctx, volume.Containers)
	} else {
		state.Containers = types.ListNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Volumes cannot be updated - all changes require replacement
	resp.Diagnostics.AddError("update not supported", "Volumes cannot be updated in place. All changes require replacement.")
}

func (r *VolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state volumeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteVolume(ctx, state.EnvironmentID.ValueString(), state.Name.ValueString()); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete volume failed", err.Error())
	}
}

func (r *VolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: environment_id/volume_name
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("invalid import ID", "Expected format: environment_id/volume_name")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}
