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

var _ resource.Resource = &VolumeBackupResource{}
var _ resource.ResourceWithImportState = &VolumeBackupResource{}

type VolumeBackupResource struct{ client *sdkclient.Client }

func NewVolumeBackupResource() resource.Resource { return &VolumeBackupResource{} }

func (r *VolumeBackupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_backup"
}

func (r *VolumeBackupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Creates and manages snapshots (backups) for a Docker volume.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:      true,
				Description:   "Backup ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id": resourceschema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"volume_name": resourceschema.StringAttribute{
				Required:    true,
				Description: "Volume name to back up",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"destination": resourceschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Where the backup is stored: \"local\" (server default) or \"s3\" (needs s3_destination_id). Changing it forces a new backup.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy_id": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Backup policy whose settings are used for this backup. Changing it forces a new backup.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"s3_destination_id": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Saved S3 destination for a remote backup. Changing it forces a new backup.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Backup run status (running, succeeded, failed).",
			},
			"error": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Error of a failed backup run.",
			},
			"format": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Storage format (rustic snapshot or legacy tar.gz).",
			},
			"trigger": resourceschema.StringAttribute{
				Computed:    true,
				Description: "How the backup was started.",
			},
			"s3_destination_name": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Name of the S3 destination used.",
			},
			"remote_available": resourceschema.BoolAttribute{
				Computed:    true,
				Description: "Whether the backup is available in the S3 destination.",
			},
			"size": resourceschema.Int64Attribute{
				Computed:    true,
				Description: "Backup size in bytes",
			},
			"created_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
			"updated_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
			},
		},
	}
}

func (r *VolumeBackupResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type volumeBackupModel struct {
	ID                types.String `tfsdk:"id"`
	EnvironmentID     types.String `tfsdk:"environment_id"`
	VolumeName        types.String `tfsdk:"volume_name"`
	Size              types.Int64  `tfsdk:"size"`
	Destination       types.String `tfsdk:"destination"`
	PolicyID          types.String `tfsdk:"policy_id"`
	S3DestinationID   types.String `tfsdk:"s3_destination_id"`
	Status            types.String `tfsdk:"status"`
	Error             types.String `tfsdk:"error"`
	Format            types.String `tfsdk:"format"`
	Trigger           types.String `tfsdk:"trigger"`
	S3DestinationName types.String `tfsdk:"s3_destination_name"`
	RemoteAvailable   types.Bool   `tfsdk:"remote_available"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (r *VolumeBackupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan volumeBackupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var body *sdkclient.CreateVolumeBackupRequest
	if isSetString(plan.Destination) || isSetString(plan.PolicyID) || isSetString(plan.S3DestinationID) {
		body = &sdkclient.CreateVolumeBackupRequest{
			Destination:     plan.Destination.ValueString(),
			PolicyID:        plan.PolicyID.ValueString(),
			S3DestinationID: plan.S3DestinationID.ValueString(),
		}
	}
	out, err := r.client.CreateVolumeBackup(ctx, plan.EnvironmentID.ValueString(), plan.VolumeName.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("create volume backup failed", err.Error())
		return
	}

	state := plan
	state.ID = types.StringValue(out.ID)
	state.VolumeName = types.StringValue(out.VolumeName)
	state.Size = types.Int64Value(out.Size)
	applyVolumeBackup(&state, out)
	state.CreatedAt = types.StringValue(out.CreatedAt)
	if out.UpdatedAt != nil {
		state.UpdatedAt = types.StringValue(*out.UpdatedAt)
	} else {
		state.UpdatedAt = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VolumeBackupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state volumeBackupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backups, err := r.client.ListVolumeBackups(ctx, state.EnvironmentID.ValueString(), state.VolumeName.ValueString())
	if err != nil {
		if r.client.IsResourceGone(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read volume backup failed", err.Error())
		return
	}

	var found *sdkclient.VolumeBackup
	for i := range backups {
		if backups[i].ID == state.ID.ValueString() {
			found = &backups[i]
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.VolumeName = types.StringValue(found.VolumeName)
	state.Size = types.Int64Value(found.Size)
	applyVolumeBackup(&state, found)
	state.CreatedAt = types.StringValue(found.CreatedAt)
	if found.UpdatedAt != nil {
		state.UpdatedAt = types.StringValue(*found.UpdatedAt)
	} else {
		state.UpdatedAt = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VolumeBackupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All mutable fields are marked RequiresReplace.
	var state volumeBackupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VolumeBackupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state volumeBackupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteVolumeBackup(ctx, state.EnvironmentID.ValueString(), state.ID.ValueString()); err != nil {
		if r.client.IsResourceGone(err) {
			return
		}
		resp.Diagnostics.AddError("delete volume backup failed", err.Error())
	}
}

func (r *VolumeBackupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: environment_id/volume_name/backup_id
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("invalid import ID", "Expected format: environment_id/volume_name/backup_id")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("volume_name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

// applyVolumeBackup copies the run details into state. policy_id and
// s3_destination_id are inputs and keep their configured value.
func applyVolumeBackup(state *volumeBackupModel, b *sdkclient.VolumeBackup) {
	state.Destination = types.StringValue(b.Destination)
	state.Status = types.StringValue(b.Status)
	state.Error = nullableString(b.Error)
	state.Format = nullableString(b.Format)
	state.Trigger = nullableString(b.Trigger)
	state.S3DestinationName = nullableString(b.S3DestinationName)
	state.RemoteAvailable = types.BoolValue(b.RemoteAvailable)
}
