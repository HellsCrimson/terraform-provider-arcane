package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &GitRepositoryBranchesDataSource{}

type GitRepositoryBranchesDataSource struct{ client *sdkclient.Client }

func NewGitRepositoryBranchesDataSource() datasource.DataSource {
	return &GitRepositoryBranchesDataSource{}
}

func (d *GitRepositoryBranchesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_git_repository_branches"
}

func (d *GitRepositoryBranchesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"repository_id":  schema.StringAttribute{Required: true},
		"branches":       schema.ListAttribute{Computed: true, ElementType: types.StringType},
		"default_branch": schema.StringAttribute{Computed: true},
	}}
}

func (d *GitRepositoryBranchesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type gitRepositoryBranchesModel struct {
	RepositoryID  types.String `tfsdk:"repository_id"`
	Branches      types.List   `tfsdk:"branches"`
	DefaultBranch types.String `tfsdk:"default_branch"`
}

func (d *GitRepositoryBranchesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitRepositoryBranchesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, err := d.client.ListGitRepositoryBranches(ctx, state.RepositoryID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to read git repository branches", err.Error())
		return
	}
	branches := make([]string, 0, len(items))
	defaultBranch := ""
	for _, b := range items {
		branches = append(branches, b.Name)
		if b.IsDefault {
			defaultBranch = b.Name
		}
	}
	state.Branches = stringsToList(ctx, branches)
	if defaultBranch != "" {
		state.DefaultBranch = types.StringValue(defaultBranch)
	} else {
		state.DefaultBranch = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
