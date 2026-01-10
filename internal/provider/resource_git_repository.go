package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &GitRepositoryResource{}
var _ resource.ResourceWithImportState = &GitRepositoryResource{}

type GitRepositoryResource struct {
	client *sdkclient.Client
}

func NewGitRepositoryResource() resource.Resource {
	return &GitRepositoryResource{}
}

// mapAuthTypeToAPI converts user-friendly auth type to API format
func mapAuthTypeToAPI(authType string) string {
	switch authType {
	case "token":
		return "http"
	default:
		return authType
	}
}

// mapAuthTypeFromAPI converts API auth type to user-friendly format
func mapAuthTypeFromAPI(authType string) string {
	switch authType {
	case "http":
		return "token"
	default:
		return authType
	}
}

func (r *GitRepositoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_git_repository"
}

func (r *GitRepositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Git repository ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": resourceschema.StringAttribute{
				Required:    true,
				Description: "Repository name",
			},
			"url": resourceschema.StringAttribute{
				Required:    true,
				Description: "Git repository URL",
			},
			"auth_type": resourceschema.StringAttribute{
				Required:    true,
				Description: "Authentication type (e.g., ssh, token, none)",
			},
			"description": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Repository description",
			},
			"enabled": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Whether the repository is enabled",
			},
			"ssh_key": resourceschema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "SSH private key for authentication",
			},
			"token": resourceschema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Access token for authentication",
			},
			"username": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Username for authentication",
			},
			"created_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *GitRepositoryResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type gitRepositoryModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	URL         types.String `tfsdk:"url"`
	AuthType    types.String `tfsdk:"auth_type"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	SSHKey      types.String `tfsdk:"ssh_key"`
	Token       types.String `tfsdk:"token"`
	Username    types.String `tfsdk:"username"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *GitRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gitRepositoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.GitRepositoryCreateRequest{
		Name:     plan.Name.ValueString(),
		URL:      plan.URL.ValueString(),
		AuthType: mapAuthTypeToAPI(plan.AuthType.ValueString()),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		body.Description = &v
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		body.Enabled = &v
	}
	if !plan.SSHKey.IsNull() && !plan.SSHKey.IsUnknown() && plan.SSHKey.ValueString() != "" {
		v := plan.SSHKey.ValueString()
		body.SSHKey = &v
	}
	if !plan.Token.IsNull() && !plan.Token.IsUnknown() && plan.Token.ValueString() != "" {
		v := plan.Token.ValueString()
		body.Token = &v
	}
	if !plan.Username.IsNull() && !plan.Username.IsUnknown() && plan.Username.ValueString() != "" {
		v := plan.Username.ValueString()
		body.Username = &v
	}

	repo, err := r.client.CreateGitRepository(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("create git repository failed", err.Error())
		return
	}

	state := gitRepositoryModel{
		ID:        types.StringValue(repo.ID),
		Name:      types.StringValue(repo.Name),
		URL:       types.StringValue(repo.URL),
		AuthType:  types.StringValue(mapAuthTypeFromAPI(repo.AuthType)),
		Enabled:   types.BoolValue(repo.Enabled),
		CreatedAt: types.StringValue(repo.CreatedAt),
		UpdatedAt: types.StringValue(repo.UpdatedAt),
		SSHKey:    plan.SSHKey,
		Token:     plan.Token,
	}

	// Handle optional fields that may be empty strings from API
	if repo.Description != "" {
		state.Description = types.StringValue(repo.Description)
	} else {
		state.Description = plan.Description
	}
	if repo.Username != "" {
		state.Username = types.StringValue(repo.Username)
	} else {
		state.Username = plan.Username
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GitRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gitRepositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repo, err := r.client.GetGitRepository(ctx, state.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read git repository failed", err.Error())
		return
	}

	state.Name = types.StringValue(repo.Name)
	state.URL = types.StringValue(repo.URL)
	state.AuthType = types.StringValue(mapAuthTypeFromAPI(repo.AuthType))
	state.Enabled = types.BoolValue(repo.Enabled)
	// Leave created_at and updated_at unchanged to avoid plan inconsistency

	// Handle optional fields that may be empty strings from API
	if repo.Description != "" {
		state.Description = types.StringValue(repo.Description)
	} else if state.Description.IsNull() {
		state.Description = types.StringNull()
	}
	// Keep existing description if API returns empty and we had a value

	if repo.Username != "" {
		state.Username = types.StringValue(repo.Username)
	} else if state.Username.IsNull() {
		state.Username = types.StringNull()
	}
	// Keep existing username if API returns empty and we had a value

	// Preserve sensitive fields from state
	// SSHKey and Token remain unchanged from state

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GitRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan gitRepositoryModel
	var state gitRepositoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.GitRepositoryUpdateRequest{}

	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		v := plan.Name.ValueString()
		body.Name = &v
	}
	if !plan.URL.IsNull() && !plan.URL.IsUnknown() {
		v := plan.URL.ValueString()
		body.URL = &v
	}
	if !plan.AuthType.IsNull() && !plan.AuthType.IsUnknown() {
		v := mapAuthTypeToAPI(plan.AuthType.ValueString())
		body.AuthType = &v
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		body.Description = &v
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		body.Enabled = &v
	}
	if !plan.SSHKey.IsNull() && !plan.SSHKey.IsUnknown() && plan.SSHKey.ValueString() != "" {
		v := plan.SSHKey.ValueString()
		body.SSHKey = &v
	}
	if !plan.Token.IsNull() && !plan.Token.IsUnknown() && plan.Token.ValueString() != "" {
		v := plan.Token.ValueString()
		body.Token = &v
	}
	if !plan.Username.IsNull() && !plan.Username.IsUnknown() && plan.Username.ValueString() != "" {
		v := plan.Username.ValueString()
		body.Username = &v
	}

	repo, err := r.client.UpdateGitRepository(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("update git repository failed", err.Error())
		return
	}

	state.Name = types.StringValue(repo.Name)
	state.URL = types.StringValue(repo.URL)
	state.AuthType = types.StringValue(mapAuthTypeFromAPI(repo.AuthType))
	state.Enabled = types.BoolValue(repo.Enabled)
	// Leave created_at and updated_at unchanged to avoid plan inconsistency

	// Handle optional fields that may be empty strings from API
	if repo.Description != "" {
		state.Description = types.StringValue(repo.Description)
	} else {
		state.Description = plan.Description
	}

	if repo.Username != "" {
		state.Username = types.StringValue(repo.Username)
	} else {
		state.Username = plan.Username
	}

	// Preserve sensitive fields from plan
	state.SSHKey = plan.SSHKey
	state.Token = plan.Token

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GitRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gitRepositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteGitRepository(ctx, state.ID.ValueString()); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete git repository failed", err.Error())
	}
}

func (r *GitRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
