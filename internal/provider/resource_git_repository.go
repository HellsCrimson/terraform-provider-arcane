package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
			"ssh_host_key_verification": resourceschema.StringAttribute{
				Optional:    true,
				Description: "SSH host key verification mode",
			},
			"ssh_key_wo":         writeOnlyAttribute("ssh_key", "SSH private key for authentication."),
			"ssh_key_wo_version": writeOnlyVersionAttribute("ssh_key"),
			"token_wo":           writeOnlyAttribute("token", "Access token for authentication."),
			"token_wo_version":   writeOnlyVersionAttribute("token"),
			"signing_key_wo":     writeOnlyAttribute("signing_key", "Private key (SSH or OpenPGP) used to sign the commits Arcane pushes, for example GitOps backups."),
			"signing_key_passphrase_wo": resourceschema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				WriteOnly: true,
				Description: "Passphrase of `signing_key_wo`. Write-only: never stored in state or plan files (requires Terraform or OpenTofu 1.11+). " +
					"Sent together with `signing_key_wo` when `signing_key_wo_version` changes.",
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("signing_key_wo")),
				},
			},
			"signing_key_wo_version": writeOnlyVersionAttribute("signing_key"),
			"has_signing_key": resourceschema.BoolAttribute{
				Computed:    true,
				Description: "Whether Arcane holds a signing key for the repository.",
			},
			"git_provider": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Git forge used to build web links: github, gitlab, gitea or bitbucket. Unset, Arcane detects it from the URL host.",
				Validators:  []validator.String{stringvalidator.OneOf("github", "gitlab", "gitea", "bitbucket")},
			},
			"commit_author_name": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Author name of the commits Arcane pushes.",
			},
			"commit_author_email": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Author email of the commits Arcane pushes.",
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
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	URL                    types.String `tfsdk:"url"`
	AuthType               types.String `tfsdk:"auth_type"`
	Description            types.String `tfsdk:"description"`
	Enabled                types.Bool   `tfsdk:"enabled"`
	SSHHostKeyVerification types.String `tfsdk:"ssh_host_key_verification"`
	SSHKeyWO               types.String `tfsdk:"ssh_key_wo"`
	SSHKeyWOVersion        types.Int64  `tfsdk:"ssh_key_wo_version"`
	TokenWO                types.String `tfsdk:"token_wo"`
	TokenWOVersion         types.Int64  `tfsdk:"token_wo_version"`
	Username               types.String `tfsdk:"username"`
	SigningKeyWO           types.String `tfsdk:"signing_key_wo"`
	SigningKeyPassphraseWO types.String `tfsdk:"signing_key_passphrase_wo"`
	SigningKeyWOVersion    types.Int64  `tfsdk:"signing_key_wo_version"`
	HasSigningKey          types.Bool   `tfsdk:"has_signing_key"`
	Provider               types.String `tfsdk:"git_provider"`
	CommitAuthorName       types.String `tfsdk:"commit_author_name"`
	CommitAuthorEmail      types.String `tfsdk:"commit_author_email"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UpdatedAt              types.String `tfsdk:"updated_at"`
}

func (r *GitRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config gitRepositoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
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
	if !plan.SSHHostKeyVerification.IsNull() && !plan.SSHHostKeyVerification.IsUnknown() && plan.SSHHostKeyVerification.ValueString() != "" {
		v := plan.SSHHostKeyVerification.ValueString()
		body.SSHHostKeyVerification = &v
	}
	// Write-only: the plan holds null, the value is only in the configuration.
	if isSetString(config.SSHKeyWO) {
		v := config.SSHKeyWO.ValueString()
		body.SSHKey = &v
	}
	if isSetString(config.TokenWO) {
		v := config.TokenWO.ValueString()
		body.Token = &v
	}
	if !plan.Username.IsNull() && !plan.Username.IsUnknown() && plan.Username.ValueString() != "" {
		v := plan.Username.ValueString()
		body.Username = &v
	}
	if isSetString(config.SigningKeyWO) {
		v := config.SigningKeyWO.ValueString()
		body.SigningKey = &v
		if isSetString(config.SigningKeyPassphraseWO) {
			p := config.SigningKeyPassphraseWO.ValueString()
			body.SigningKeyPassphrase = &p
		}
	}
	body.Provider = optionalString(plan.Provider)
	body.CommitAuthorName = optionalString(plan.CommitAuthorName)
	body.CommitAuthorEmail = optionalString(plan.CommitAuthorEmail)

	repo, err := r.client.CreateGitRepository(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("create git repository failed", err.Error())
		return
	}

	state := gitRepositoryModel{
		ID:                     types.StringValue(repo.ID),
		Name:                   types.StringValue(repo.Name),
		URL:                    types.StringValue(repo.URL),
		AuthType:               types.StringValue(mapAuthTypeFromAPI(repo.AuthType)),
		Enabled:                plan.Enabled,
		CreatedAt:              types.StringValue(repo.CreatedAt),
		UpdatedAt:              types.StringValue(repo.UpdatedAt),
		SSHHostKeyVerification: plan.SSHHostKeyVerification,
		SSHKeyWOVersion:        plan.SSHKeyWOVersion,
		TokenWOVersion:         plan.TokenWOVersion,
		Description:            plan.Description,
		Username:               plan.Username,
		SigningKeyWOVersion:    plan.SigningKeyWOVersion,
	}
	state.HasSigningKey = types.BoolValue(repo.HasSigningKey)
	state.Provider, state.CommitAuthorName, state.CommitAuthorEmail = gitRepositoryExtras(repo, plan)

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
	if !state.Enabled.IsNull() && !state.Enabled.IsUnknown() {
		state.Enabled = types.BoolValue(repo.Enabled)
	}
	if !state.SSHHostKeyVerification.IsNull() && !state.SSHHostKeyVerification.IsUnknown() && repo.SSHHostKeyVerification != "" {
		state.SSHHostKeyVerification = types.StringValue(repo.SSHHostKeyVerification)
	}
	// Leave created_at and updated_at unchanged to avoid plan inconsistency

	// Handle optional fields that may be empty strings from API
	if !state.Description.IsNull() && !state.Description.IsUnknown() && repo.Description != "" {
		state.Description = types.StringValue(repo.Description)
	}
	// Keep existing description if API returns empty and we had a value

	if !state.Username.IsNull() && !state.Username.IsUnknown() && repo.Username != "" {
		state.Username = types.StringValue(repo.Username)
	}
	// Keep existing username if API returns empty and we had a value

	state.HasSigningKey = types.BoolValue(repo.HasSigningKey)
	state.Provider, state.CommitAuthorName, state.CommitAuthorEmail = gitRepositoryExtras(repo, state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GitRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config gitRepositoryModel
	var state gitRepositoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
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
	if !plan.SSHHostKeyVerification.IsNull() && !plan.SSHHostKeyVerification.IsUnknown() && plan.SSHHostKeyVerification.ValueString() != "" {
		v := plan.SSHHostKeyVerification.ValueString()
		body.SSHHostKeyVerification = &v
	}
	// Write-only credentials are only re-sent when their version changes.
	if writeOnlySend(config.SSHKeyWO, plan.SSHKeyWOVersion, state.SSHKeyWOVersion) {
		v := config.SSHKeyWO.ValueString()
		body.SSHKey = &v
	}
	if writeOnlySend(config.TokenWO, plan.TokenWOVersion, state.TokenWOVersion) {
		v := config.TokenWO.ValueString()
		body.Token = &v
	}
	if !plan.Username.IsNull() && !plan.Username.IsUnknown() && plan.Username.ValueString() != "" {
		v := plan.Username.ValueString()
		body.Username = &v
	}
	if writeOnlySend(config.SigningKeyWO, plan.SigningKeyWOVersion, state.SigningKeyWOVersion) {
		v := config.SigningKeyWO.ValueString()
		body.SigningKey = &v
		p := ""
		if isSetString(config.SigningKeyPassphraseWO) {
			p = config.SigningKeyPassphraseWO.ValueString()
		}
		body.SigningKeyPassphrase = &p
	}
	// Removed optional strings are cleared with an empty string.
	body.Provider = clearableString(plan.Provider, state.Provider)
	body.CommitAuthorName = clearableString(plan.CommitAuthorName, state.CommitAuthorName)
	body.CommitAuthorEmail = clearableString(plan.CommitAuthorEmail, state.CommitAuthorEmail)

	repo, err := r.client.UpdateGitRepository(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("update git repository failed", err.Error())
		return
	}

	state.Name = types.StringValue(repo.Name)
	state.URL = types.StringValue(repo.URL)
	state.AuthType = types.StringValue(mapAuthTypeFromAPI(repo.AuthType))
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		state.Enabled = types.BoolValue(repo.Enabled)
	} else {
		state.Enabled = plan.Enabled
	}
	if !plan.SSHHostKeyVerification.IsNull() && !plan.SSHHostKeyVerification.IsUnknown() && repo.SSHHostKeyVerification != "" {
		state.SSHHostKeyVerification = types.StringValue(repo.SSHHostKeyVerification)
	} else {
		state.SSHHostKeyVerification = plan.SSHHostKeyVerification
	}
	// Leave created_at and updated_at unchanged to avoid plan inconsistency

	// Handle optional fields that may be empty strings from API
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() && repo.Description != "" {
		state.Description = types.StringValue(repo.Description)
	} else {
		state.Description = plan.Description
	}

	if !plan.Username.IsNull() && !plan.Username.IsUnknown() && repo.Username != "" {
		state.Username = types.StringValue(repo.Username)
	} else {
		state.Username = plan.Username
	}

	// Preserve sensitive fields from plan
	state.SSHKeyWOVersion = plan.SSHKeyWOVersion
	state.TokenWOVersion = plan.TokenWOVersion
	state.SigningKeyWOVersion = plan.SigningKeyWOVersion
	state.HasSigningKey = types.BoolValue(repo.HasSigningKey)
	state.Provider, state.CommitAuthorName, state.CommitAuthorEmail = gitRepositoryExtras(repo, plan)

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

// gitRepositoryExtras resolves the forge and commit author settings for
// state: a configured value is echoed from the API, an unset one stays null.
func gitRepositoryExtras(repo *sdkclient.GitRepository, desired gitRepositoryModel) (provider, authorName, authorEmail types.String) {
	return echoOptionalString(desired.Provider, repo.Provider),
		echoOptionalString(desired.CommitAuthorName, repo.CommitAuthorName),
		echoOptionalString(desired.CommitAuthorEmail, repo.CommitAuthorEmail)
}
