package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &JobsDataSource{}

type JobsDataSource struct{ client *sdkclient.Client }

func NewJobsDataSource() datasource.DataSource { return &JobsDataSource{} }

func (d *JobsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jobs"
}

func (d *JobsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"environment_id": schema.StringAttribute{Required: true},
		"is_agent":       schema.BoolAttribute{Computed: true},
		"total_count":    schema.Int64Attribute{Computed: true},
		"jobs_json":      schema.StringAttribute{Computed: true},
	}}
}

func (d *JobsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type jobsModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	IsAgent       types.Bool   `tfsdk:"is_agent"`
	TotalCount    types.Int64  `tfsdk:"total_count"`
	JobsJSON      types.String `tfsdk:"jobs_json"`
}

func (d *JobsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state jobsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := d.client.ListJobs(ctx, state.EnvironmentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to list jobs", err.Error())
		return
	}
	state.IsAgent = types.BoolValue(out.IsAgent)
	state.TotalCount = types.Int64Value(int64(len(out.Jobs)))
	state.JobsJSON = types.StringValue(mustJSON(out.Jobs))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
