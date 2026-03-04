package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VolumeBackupHasPathDataSource{}

type VolumeBackupHasPathDataSource struct{ client *sdkclient.Client }

func NewVolumeBackupHasPathDataSource() datasource.DataSource {
	return &VolumeBackupHasPathDataSource{}
}

func (d *VolumeBackupHasPathDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_backup_has_path"
}

func (d *VolumeBackupHasPathDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"environment_id": schema.StringAttribute{Required: true},
		"backup_id":      schema.StringAttribute{Required: true},
		"path":           schema.StringAttribute{Required: true},
		"exists":         schema.BoolAttribute{Computed: true},
	}}
}

func (d *VolumeBackupHasPathDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type volumeBackupHasPathModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	BackupID      types.String `tfsdk:"backup_id"`
	Path          types.String `tfsdk:"path"`
	Exists        types.Bool   `tfsdk:"exists"`
}

func (d *VolumeBackupHasPathDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state volumeBackupHasPathModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	exists, err := d.client.VolumeBackupHasPath(ctx, state.EnvironmentID.ValueString(), state.BackupID.ValueString(), state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to check backup path", err.Error())
		return
	}
	state.Exists = types.BoolValue(exists)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
