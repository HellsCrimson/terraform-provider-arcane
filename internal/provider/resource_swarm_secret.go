package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SwarmSecretResource{}
var _ resource.ResourceWithImportState = &SwarmSecretResource{}

type SwarmSecretResource struct{ client *sdkclient.Client }

func NewSwarmSecretResource() resource.Resource { return &SwarmSecretResource{} }

func (r *SwarmSecretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_swarm_secret"
}

func (r *SwarmSecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manages a Docker Swarm secret in an Arcane environment.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:      true,
				Description:   "Swarm secret ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id": resourceschema.StringAttribute{
				Required:    true,
				Description: "Environment ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": resourceschema.StringAttribute{
				Required:    true,
				Description: "Secret name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"data_wo": func() resourceschema.StringAttribute {
				a := writeOnlyAttribute("data", "Secret value (plaintext). The provider encodes this to base64 for the API. "+
					"Swarm secrets are immutable, so changing `data_wo_version` replaces the secret with the configured value.")
				a.Optional, a.Required = false, true
				return a
			}(),
			"data_wo_version": writeOnlyVersionAttribute("data",
				// Changing the version replaces the (immutable) secret. Setting
				// it for the first time on an existing secret (state written by a
				// release that still had the stored `data`) keeps the secret.
				int64planmodifier.RequiresReplaceIf(func(_ context.Context, req planmodifier.Int64Request, resp *int64planmodifier.RequiresReplaceIfFuncResponse) {
					resp.RequiresReplace = !req.StateValue.IsNull()
				}, "Changing data_wo_version replaces the secret, except when first set on an existing secret.", "Changing `data_wo_version` replaces the secret, except when first set on an existing secret."),
			),
			"labels": resourceschema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Secret labels",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"version_index": resourceschema.Int64Attribute{
				Computed:    true,
				Description: "Swarm object version index",
			},
			"created_at": resourceschema.StringAttribute{
				Computed:      true,
				Description:   "Creation timestamp",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
			},
		},
	}
}

func (r *SwarmSecretResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type swarmSecretModel struct {
	ID            types.String `tfsdk:"id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	Name          types.String `tfsdk:"name"`
	DataWO        types.String `tfsdk:"data_wo"`
	DataWOVersion types.Int64  `tfsdk:"data_wo_version"`
	Labels        types.Map    `tfsdk:"labels"`
	VersionIndex  types.Int64  `tfsdk:"version_index"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *SwarmSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config swarmSecretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write-only: the plan holds null, the value is only in the configuration.
	data := config.DataWO.ValueString()
	spec := sdkclient.DockerSwarmSecretSpec{
		Name: plan.Name.ValueString(),
		Data: sdkclient.EncodeSwarmSecretData(data),
	}
	if !plan.Labels.IsNull() && !plan.Labels.IsUnknown() {
		spec.Labels = mapFromStringMap(ctx, plan.Labels)
	}

	secret, err := r.client.CreateSwarmSecret(ctx, plan.EnvironmentID.ValueString(), sdkclient.SwarmSecretCreateRequest{Spec: spec})
	if err != nil {
		resp.Diagnostics.AddError("create swarm secret failed", err.Error())
		return
	}

	state := swarmSecretModel{
		ID:            types.StringValue(secret.ID),
		EnvironmentID: plan.EnvironmentID,
		Name:          types.StringValue(secret.Spec.Name),
		DataWOVersion: plan.DataWOVersion,
		Labels:        stringMapToMap(ctx, secret.Spec.Labels),
		VersionIndex:  types.Int64Value(secret.Version.Index),
		CreatedAt:     types.StringValue(secret.CreatedAt),
		UpdatedAt:     types.StringValue(secret.UpdatedAt),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SwarmSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state swarmSecretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := r.client.GetSwarmSecret(ctx, state.EnvironmentID.ValueString(), state.ID.ValueString())
	if err != nil {
		if r.client.IsResourceGone(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read swarm secret failed", err.Error())
		return
	}

	state.Name = types.StringValue(secret.Spec.Name)
	if !state.Labels.IsNull() && !state.Labels.IsUnknown() {
		state.Labels = stringMapToMap(ctx, secret.Spec.Labels)
	}
	state.VersionIndex = types.Int64Value(secret.Version.Index)
	state.CreatedAt = types.StringValue(secret.CreatedAt)
	state.UpdatedAt = types.StringValue(secret.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update only handles data_wo_version being set for the first time on an
// existing secret: every other change replaces the (immutable) secret, so
// nothing is sent to Arcane.
func (r *SwarmSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state swarmSecretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := r.client.GetSwarmSecret(ctx, state.EnvironmentID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("read swarm secret failed", err.Error())
		return
	}

	state.DataWOVersion = plan.DataWOVersion
	state.VersionIndex = types.Int64Value(secret.Version.Index)
	state.UpdatedAt = types.StringValue(secret.UpdatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SwarmSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state swarmSecretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSwarmSecret(ctx, state.EnvironmentID.ValueString(), state.ID.ValueString()); err != nil {
		if r.client.IsResourceGone(err) {
			return
		}
		resp.Diagnostics.AddError("delete swarm secret failed", err.Error())
	}
}

func (r *SwarmSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("invalid import id", "expected env_id:secret_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
