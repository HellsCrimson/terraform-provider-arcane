package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &FederatedCredentialResource{}
var _ resource.ResourceWithImportState = &FederatedCredentialResource{}

type FederatedCredentialResource struct {
	client *sdkclient.Client
}

func NewFederatedCredentialResource() resource.Resource { return &FederatedCredentialResource{} }

func (r *FederatedCredentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_federated_credential"
}

func (r *FederatedCredentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manages a workload identity federation trust rule. Allows an external OIDC issuer " +
			"to exchange tokens for an Arcane service token bound to a role.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:      true,
				Description:   "Unique identifier of the federated credential.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": resourceschema.StringAttribute{
				Required:    true,
				Description: "Display name.",
			},
			"enabled": resourceschema.BoolAttribute{
				Required:    true,
				Description: "Whether token exchanges are allowed.",
			},
			"issuer_url": resourceschema.StringAttribute{
				Required:    true,
				Description: "Trusted external OIDC issuer URL.",
			},
			"audiences": resourceschema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Allowed external token audiences (at least one).",
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
			},
			"subject_match": resourceschema.StringAttribute{
				Required:    true,
				Description: "Exact subject or anchored glob pattern to match.",
			},
			"role_id": resourceschema.StringAttribute{
				Required:    true,
				Description: "Mapped role ID granted to exchanged tokens.",
			},
			"description": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Optional description.",
			},
			"environment_id": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Optional environment scope for the role assignment.",
			},
			"expires_at": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Optional credential expiration (RFC3339).",
			},
			"match_type": resourceschema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Subject match strategy: 'exact' or 'glob'.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"subject_claim": resourceschema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Claim path to match against; defaults to 'sub'.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"token_ttl_seconds": resourceschema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Issued token lifetime in seconds (60-3600).",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"role_name": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Mapped role name.",
			},
			"environment_name": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Mapped environment name when scoped.",
			},
			"identity_user_id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Dedicated service user ID backing issued tokens.",
			},
			"service_username": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Dedicated service account username.",
			},
			"last_used_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Last successful token exchange.",
			},
			"created_at": resourceschema.StringAttribute{
				Computed:      true,
				Description:   "Creation timestamp.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp.",
			},
		},
	}
}

func (r *FederatedCredentialResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type federatedCredentialModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	IssuerURL       types.String `tfsdk:"issuer_url"`
	Audiences       types.Set    `tfsdk:"audiences"`
	SubjectMatch    types.String `tfsdk:"subject_match"`
	RoleID          types.String `tfsdk:"role_id"`
	Description     types.String `tfsdk:"description"`
	EnvironmentID   types.String `tfsdk:"environment_id"`
	ExpiresAt       types.String `tfsdk:"expires_at"`
	MatchType       types.String `tfsdk:"match_type"`
	SubjectClaim    types.String `tfsdk:"subject_claim"`
	TokenTTLSeconds types.Int64  `tfsdk:"token_ttl_seconds"`
	RoleName        types.String `tfsdk:"role_name"`
	EnvironmentName types.String `tfsdk:"environment_name"`
	IdentityUserID  types.String `tfsdk:"identity_user_id"`
	ServiceUsername types.String `tfsdk:"service_username"`
	LastUsedAt      types.String `tfsdk:"last_used_at"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

func (r *FederatedCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan federatedCredentialModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	auds, diags := roleSetToStringSlice(ctx, plan.Audiences)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := sdkclient.FederatedCredentialCreateRequest{
		Name:         plan.Name.ValueString(),
		Enabled:      plan.Enabled.ValueBool(),
		IssuerURL:    plan.IssuerURL.ValueString(),
		Audiences:    auds,
		SubjectMatch: plan.SubjectMatch.ValueString(),
		RoleID:       plan.RoleID.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		body.Description = &v
	}
	if !plan.EnvironmentID.IsNull() && !plan.EnvironmentID.IsUnknown() && plan.EnvironmentID.ValueString() != "" {
		v := plan.EnvironmentID.ValueString()
		body.EnvironmentID = &v
	}
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() && plan.ExpiresAt.ValueString() != "" {
		v := plan.ExpiresAt.ValueString()
		body.ExpiresAt = &v
	}
	if !plan.MatchType.IsNull() && !plan.MatchType.IsUnknown() {
		v := plan.MatchType.ValueString()
		body.MatchType = &v
	}
	if !plan.SubjectClaim.IsNull() && !plan.SubjectClaim.IsUnknown() {
		v := plan.SubjectClaim.ValueString()
		body.SubjectClaim = &v
	}
	if !plan.TokenTTLSeconds.IsNull() && !plan.TokenTTLSeconds.IsUnknown() {
		v := plan.TokenTTLSeconds.ValueInt64()
		body.TokenTTLSeconds = &v
	}

	fc, err := r.client.CreateFederatedCredential(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("create federated credential failed", err.Error())
		return
	}

	state, diags := buildFederatedCredentialState(ctx, fc, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FederatedCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state federatedCredentialModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fc, err := r.client.GetFederatedCredential(ctx, state.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read federated credential failed", err.Error())
		return
	}

	newState, diags := buildFederatedCredentialState(ctx, fc, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Leave created_at/updated_at as stored to avoid plan inconsistency.
	newState.CreatedAt = state.CreatedAt
	if !state.Description.IsNull() && fc.Description == "" {
		newState.Description = state.Description
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *FederatedCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan federatedCredentialModel
	var state federatedCredentialModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	auds, diags := roleSetToStringSlice(ctx, plan.Audiences)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := plan.Name.ValueString()
	enabled := plan.Enabled.ValueBool()
	issuer := plan.IssuerURL.ValueString()
	subjectMatch := plan.SubjectMatch.ValueString()
	roleID := plan.RoleID.ValueString()
	body := sdkclient.FederatedCredentialUpdateRequest{
		Name:         &name,
		Enabled:      &enabled,
		IssuerURL:    &issuer,
		Audiences:    auds,
		SubjectMatch: &subjectMatch,
		RoleID:       &roleID,
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		body.Description = &v
	}
	if !plan.EnvironmentID.IsNull() && !plan.EnvironmentID.IsUnknown() && plan.EnvironmentID.ValueString() != "" {
		v := plan.EnvironmentID.ValueString()
		body.EnvironmentID = &v
	}
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() && plan.ExpiresAt.ValueString() != "" {
		v := plan.ExpiresAt.ValueString()
		body.ExpiresAt = &v
	}
	if !plan.MatchType.IsNull() && !plan.MatchType.IsUnknown() {
		v := plan.MatchType.ValueString()
		body.MatchType = &v
	}
	if !plan.SubjectClaim.IsNull() && !plan.SubjectClaim.IsUnknown() {
		v := plan.SubjectClaim.ValueString()
		body.SubjectClaim = &v
	}
	if !plan.TokenTTLSeconds.IsNull() && !plan.TokenTTLSeconds.IsUnknown() {
		v := plan.TokenTTLSeconds.ValueInt64()
		body.TokenTTLSeconds = &v
	}

	fc, err := r.client.UpdateFederatedCredential(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("update federated credential failed", err.Error())
		return
	}

	newState, diags := buildFederatedCredentialState(ctx, fc, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	newState.CreatedAt = state.CreatedAt
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *FederatedCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state federatedCredentialModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteFederatedCredential(ctx, state.ID.ValueString()); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete federated credential failed", err.Error())
	}
}

func (r *FederatedCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildFederatedCredentialState(ctx context.Context, fc *sdkclient.FederatedCredential, plan federatedCredentialModel) (federatedCredentialModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	audSet, d := roleStringSliceToSet(ctx, fc.Audiences)
	diags.Append(d...)
	state := federatedCredentialModel{
		ID:              types.StringValue(fc.ID),
		Name:            types.StringValue(fc.Name),
		Enabled:         types.BoolValue(fc.Enabled),
		IssuerURL:       types.StringValue(fc.IssuerURL),
		Audiences:       audSet,
		SubjectMatch:    types.StringValue(fc.SubjectMatch),
		RoleID:          types.StringValue(fc.RoleID),
		MatchType:       stringOrNullEmpty(fc.MatchType),
		SubjectClaim:    stringOrNullEmpty(fc.SubjectClaim),
		TokenTTLSeconds: types.Int64Value(fc.TokenTTLSeconds),
		RoleName:        stringOrNullEmpty(fc.RoleName),
		EnvironmentName: stringOrNullEmpty(fc.EnvironmentName),
		IdentityUserID:  stringOrNullEmpty(fc.IdentityUserID),
		ServiceUsername: stringOrNullEmpty(fc.ServiceUsername),
		LastUsedAt:      stringOrNullEmpty(fc.LastUsedAt),
		CreatedAt:       stringOrNullEmpty(fc.CreatedAt),
		UpdatedAt:       stringOrNullEmpty(fc.UpdatedAt),
	}
	// Preserve optional config values not always echoed identically by the API.
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		state.Description = plan.Description
	} else {
		state.Description = stringOrNullEmpty(fc.Description)
	}
	if !plan.EnvironmentID.IsNull() && !plan.EnvironmentID.IsUnknown() {
		state.EnvironmentID = plan.EnvironmentID
	} else {
		state.EnvironmentID = stringOrNullEmpty(fc.EnvironmentID)
	}
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		state.ExpiresAt = plan.ExpiresAt
	} else {
		state.ExpiresAt = stringOrNullEmpty(fc.ExpiresAt)
	}
	return state, diags
}
