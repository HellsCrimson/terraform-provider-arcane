package provider

import (
	"context"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &GitRepositoryFilesDataSource{}

type GitRepositoryFilesDataSource struct{ client *sdkclient.Client }

func NewGitRepositoryFilesDataSource() datasource.DataSource { return &GitRepositoryFilesDataSource{} }

func (d *GitRepositoryFilesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_git_repository_files"
}

func (d *GitRepositoryFilesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"repository_id": schema.StringAttribute{Required: true},
		"branch":        schema.StringAttribute{Optional: true},
		"path":          schema.StringAttribute{Optional: true},
		"current_path":  schema.StringAttribute{Computed: true},
		"files_json":    schema.StringAttribute{Computed: true},
	}}
}

func (d *GitRepositoryFilesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = setProviderClient(req, resp)
}

type gitRepositoryFilesModel struct {
	RepositoryID types.String `tfsdk:"repository_id"`
	Branch       types.String `tfsdk:"branch"`
	Path         types.String `tfsdk:"path"`
	CurrentPath  types.String `tfsdk:"current_path"`
	FilesJSON    types.String `tfsdk:"files_json"`
}

func (d *GitRepositoryFilesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state gitRepositoryFilesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	branch := ""
	if !state.Branch.IsNull() && !state.Branch.IsUnknown() {
		branch = state.Branch.ValueString()
	}
	browsePath := ""
	if !state.Path.IsNull() && !state.Path.IsUnknown() {
		browsePath = state.Path.ValueString()
	}
	out, err := d.client.BrowseGitRepositoryFiles(ctx, state.RepositoryID.ValueString(), branch, browsePath)
	if err != nil {
		resp.Diagnostics.AddError("failed to browse git repository files", err.Error())
		return
	}
	state.CurrentPath = types.StringValue(out.Path)
	state.FilesJSON = types.StringValue(mustJSON(out.Files))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
