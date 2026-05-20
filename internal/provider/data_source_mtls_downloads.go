package provider

import (
	"context"
	"encoding/base64"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &EdgeMTLSCADataSource{}
var _ datasource.DataSource = &EnvironmentMTLSBundleDataSource{}
var _ datasource.DataSource = &EnvironmentMTLSFileDataSource{}

type EdgeMTLSCADataSource struct {
	client *sdkclient.Client
}

func NewEdgeMTLSCADataSource() datasource.DataSource { return &EdgeMTLSCADataSource{} }

func (d *EdgeMTLSCADataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_mtls_ca"
}

func (d *EdgeMTLSCADataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Downloads the Arcane-managed edge mTLS certificate authority.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Computed: true},
			"content":        schema.StringAttribute{Computed: true},
			"content_base64": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *EdgeMTLSCADataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type edgeMTLSCAModel struct {
	ID            types.String `tfsdk:"id"`
	Content       types.String `tfsdk:"content"`
	ContentBase64 types.String `tfsdk:"content_base64"`
}

func (d *EdgeMTLSCADataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state edgeMTLSCAModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	b, err := d.client.DownloadEdgeMTLSCA(ctx)
	if err != nil {
		resp.Diagnostics.AddError("failed to download edge mTLS CA", err.Error())
		return
	}
	state.ID = types.StringValue("edge-mtls-ca")
	state.Content = types.StringValue(string(b))
	state.ContentBase64 = types.StringValue(base64.StdEncoding.EncodeToString(b))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type EnvironmentMTLSBundleDataSource struct {
	client *sdkclient.Client
}

func NewEnvironmentMTLSBundleDataSource() datasource.DataSource {
	return &EnvironmentMTLSBundleDataSource{}
}

func (d *EnvironmentMTLSBundleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_mtls_bundle"
}

func (d *EnvironmentMTLSBundleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Downloads the generated mTLS client certificate bundle for an edge environment.",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{Required: true},
			"id":             schema.StringAttribute{Computed: true},
			"content_base64": schema.StringAttribute{Computed: true, Sensitive: true},
		},
	}
}

func (d *EnvironmentMTLSBundleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type environmentMTLSBundleModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	ID            types.String `tfsdk:"id"`
	ContentBase64 types.String `tfsdk:"content_base64"`
}

func (d *EnvironmentMTLSBundleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state environmentMTLSBundleModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := state.EnvironmentID.ValueString()
	b, err := d.client.DownloadEnvironmentMTLSBundle(ctx, envID)
	if err != nil {
		resp.Diagnostics.AddError("failed to download environment mTLS bundle", err.Error())
		return
	}
	state.ID = types.StringValue(envID + ":mtls-bundle")
	state.ContentBase64 = types.StringValue(base64.StdEncoding.EncodeToString(b))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type EnvironmentMTLSFileDataSource struct {
	client *sdkclient.Client
}

func NewEnvironmentMTLSFileDataSource() datasource.DataSource {
	return &EnvironmentMTLSFileDataSource{}
}

func (d *EnvironmentMTLSFileDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_mtls_file"
}

func (d *EnvironmentMTLSFileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Downloads an individual generated mTLS certificate asset for an edge environment.",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{Required: true},
			"file_name":      schema.StringAttribute{Required: true},
			"id":             schema.StringAttribute{Computed: true},
			"content":        schema.StringAttribute{Computed: true, Sensitive: true},
			"content_base64": schema.StringAttribute{Computed: true, Sensitive: true},
		},
	}
}

func (d *EnvironmentMTLSFileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type environmentMTLSFileModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	FileName      types.String `tfsdk:"file_name"`
	ID            types.String `tfsdk:"id"`
	Content       types.String `tfsdk:"content"`
	ContentBase64 types.String `tfsdk:"content_base64"`
}

func (d *EnvironmentMTLSFileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state environmentMTLSFileModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := state.EnvironmentID.ValueString()
	fileName := state.FileName.ValueString()
	b, err := d.client.DownloadEnvironmentMTLSFile(ctx, envID, fileName)
	if err != nil {
		resp.Diagnostics.AddError("failed to download environment mTLS file", err.Error())
		return
	}
	state.ID = types.StringValue(envID + ":mtls-file:" + fileName)
	state.Content = types.StringValue(string(b))
	state.ContentBase64 = types.StringValue(base64.StdEncoding.EncodeToString(b))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
