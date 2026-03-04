package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ImageDataSource{}

type ImageDataSource struct{ client *sdkclient.Client }

func NewImageDataSource() datasource.DataSource { return &ImageDataSource{} }

func (d *ImageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image"
}

func (d *ImageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"environment_id": schema.StringAttribute{Required: true},
		"id":             schema.StringAttribute{Required: true},
		"created":        schema.StringAttribute{Computed: true},
		"size":           schema.Int64Attribute{Computed: true},
		"author":         schema.StringAttribute{Computed: true},
		"architecture":   schema.StringAttribute{Computed: true},
		"os":             schema.StringAttribute{Computed: true},
		"data_json":      schema.StringAttribute{Computed: true},
	}}
}

func (d *ImageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type imageModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	ID            types.String `tfsdk:"id"`
	Created       types.String `tfsdk:"created"`
	Size          types.Int64  `tfsdk:"size"`
	Author        types.String `tfsdk:"author"`
	Architecture  types.String `tfsdk:"architecture"`
	OS            types.String `tfsdk:"os"`
	DataJSON      types.String `tfsdk:"data_json"`
}

func (d *ImageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state imageModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	img, err := d.client.GetImage(ctx, state.EnvironmentID.ValueString(), state.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("image not found", "No image with id: "+state.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read image", err.Error())
		return
	}
	state.Created = types.StringValue(img.Created)
	state.Size = types.Int64Value(img.Size)
	state.Author = types.StringValue(img.Author)
	state.Architecture = types.StringValue(img.Architecture)
	state.OS = types.StringValue(img.OS)
	state.DataJSON = types.StringValue(mustJSON(img))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
