package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &GitRepositoryDataSource{}

type GitRepositoryDataSource struct {
	client *sdkclient.Client
}

func NewGitRepositoryDataSource() datasource.DataSource {
	return &GitRepositoryDataSource{}
}

func (d *GitRepositoryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_git_repository"
}

func (d *GitRepositoryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane git repository",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Git repository ID",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Repository name",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "Git repository URL",
			},
			"auth_type": schema.StringAttribute{
				Computed:    true,
				Description: "Authentication type",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Repository description",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository is enabled",
			},
			"username": schema.StringAttribute{
				Computed:    true,
				Description: "Username for authentication",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
			},
		},
	}
}

func (d *GitRepositoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type gitRepositoryDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	URL         types.String `tfsdk:"url"`
	AuthType    types.String `tfsdk:"auth_type"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Username    types.String `tfsdk:"username"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (d *GitRepositoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config gitRepositoryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repo, err := d.client.GetGitRepository(ctx, config.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("git repository not found", "No repository with id: "+config.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read git repository", err.Error())
		return
	}

	state := gitRepositoryDataSourceModel{
		ID:        types.StringValue(repo.ID),
		Name:      types.StringValue(repo.Name),
		URL:       types.StringValue(repo.URL),
		AuthType:  types.StringValue(mapAuthTypeFromAPI(repo.AuthType)),
		Enabled:   types.BoolValue(repo.Enabled),
		CreatedAt: types.StringValue(repo.CreatedAt),
		UpdatedAt: types.StringValue(repo.UpdatedAt),
	}

	if repo.Description != "" {
		state.Description = types.StringValue(repo.Description)
	} else {
		state.Description = types.StringNull()
	}
	if repo.Username != "" {
		state.Username = types.StringValue(repo.Username)
	} else {
		state.Username = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
