package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VolumeDataSource{}

type VolumeDataSource struct {
	client *sdkclient.Client
}

func NewVolumeDataSource() datasource.DataSource {
	return &VolumeDataSource{}
}

func (d *VolumeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (d *VolumeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane volume",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Volume ID",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Volume name",
			},
			"driver": schema.StringAttribute{
				Computed:    true,
				Description: "Volume driver",
			},
			"driver_opts": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Driver-specific options",
			},
			"labels": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Volume labels",
			},
			"mountpoint": schema.StringAttribute{
				Computed:    true,
				Description: "Mount point on host",
			},
			"scope": schema.StringAttribute{
				Computed:    true,
				Description: "Volume scope",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
		},
	}
}

func (d *VolumeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*sdkclient.Client)
	if !ok {
		resp.Diagnostics.AddError("unexpected provider data type", "Expected *sdkclient.Client")
		return
	}
	d.client = client
}

type volumeDataSourceModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Driver        types.String `tfsdk:"driver"`
	DriverOpts    types.Map    `tfsdk:"driver_opts"`
	Labels        types.Map    `tfsdk:"labels"`
	Mountpoint    types.String `tfsdk:"mountpoint"`
	Scope         types.String `tfsdk:"scope"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func (d *VolumeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config volumeDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	volume, err := d.client.GetVolume(ctx, config.EnvironmentID.ValueString(), config.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("volume not found", "No volume with id: "+config.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read volume", err.Error())
		return
	}

	state := volumeDataSourceModel{
		EnvironmentID: config.EnvironmentID,
		ID:            types.StringValue(volume.Name),
		Name:          types.StringValue(volume.Name),
		Driver:        types.StringValue(volume.Driver),
		Mountpoint:    types.StringValue(volume.Mountpoint),
		Scope:         types.StringValue(volume.Scope),
		CreatedAt:     types.StringValue(volume.CreatedAt),
	}

	state.DriverOpts = stringMapToMap(ctx, volume.Options)
	state.Labels = stringMapToMap(ctx, volume.Labels)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
