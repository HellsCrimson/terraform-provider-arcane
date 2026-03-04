package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VolumeBackupFilesDataSource{}

type VolumeBackupFilesDataSource struct{ client *sdkclient.Client }

func NewVolumeBackupFilesDataSource() datasource.DataSource { return &VolumeBackupFilesDataSource{} }

func (d *VolumeBackupFilesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_backup_files"
}

func (d *VolumeBackupFilesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"environment_id": schema.StringAttribute{Required: true},
		"backup_id":      schema.StringAttribute{Required: true},
		"files":          schema.ListAttribute{Computed: true, ElementType: types.StringType},
	}}
}

func (d *VolumeBackupFilesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type volumeBackupFilesModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	BackupID      types.String `tfsdk:"backup_id"`
	Files         types.List   `tfsdk:"files"`
}

func (d *VolumeBackupFilesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state volumeBackupFilesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	files, err := d.client.ListVolumeBackupFiles(ctx, state.EnvironmentID.ValueString(), state.BackupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to list volume backup files", err.Error())
		return
	}
	state.Files = stringsToList(ctx, files)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
