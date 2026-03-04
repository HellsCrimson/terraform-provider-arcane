package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &IgnoredVulnerabilitiesDataSource{}

type IgnoredVulnerabilitiesDataSource struct{ client *sdkclient.Client }

func NewIgnoredVulnerabilitiesDataSource() datasource.DataSource {
	return &IgnoredVulnerabilitiesDataSource{}
}

func (d *IgnoredVulnerabilitiesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ignored_vulnerabilities"
}

func (d *IgnoredVulnerabilitiesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"environment_id": schema.StringAttribute{Required: true},
		"total_count":    schema.Int64Attribute{Computed: true},
		"data_json":      schema.StringAttribute{Computed: true},
	}}
}

func (d *IgnoredVulnerabilitiesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type ignoredVulnerabilitiesModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	TotalCount    types.Int64  `tfsdk:"total_count"`
	DataJSON      types.String `tfsdk:"data_json"`
}

func (d *IgnoredVulnerabilitiesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ignoredVulnerabilitiesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, err := d.client.ListIgnoredVulnerabilities(ctx, state.EnvironmentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to list ignored vulnerabilities", err.Error())
		return
	}
	state.TotalCount = types.Int64Value(int64(len(items)))
	state.DataJSON = types.StringValue(mustJSON(items))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
