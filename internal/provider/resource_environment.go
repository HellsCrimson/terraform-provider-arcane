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

var _ resource.Resource = &EnvironmentResource{}
var _ resource.ResourceWithImportState = &EnvironmentResource{}

type EnvironmentResource struct{ client *sdkclient.Client }

func NewEnvironmentResource() resource.Resource { return &EnvironmentResource{} }

func (r *EnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *EnvironmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "An Arcane environment. Environments are agent-based: create the environment, " +
			"then start an Arcane agent with the enrollment_token (single use) so it joins this environment.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Environment ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Environment display name",
			},
			"api_url": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Environment URL derived by Arcane (agent://<name> for agent environments).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Whether the environment is enabled.",
			},

			// Computed fields
			"status": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Environment status (pending until an agent enrolls).",
			},
			"enrollment_token": resourceschema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Single-use enrollment token minted when the environment is created; pass it to the agent as ARCANE_AGENT_TOKEN. Not available for imported environments.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"edge_agent_instance": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Agent instance identifier.",
			},
			"edge_capabilities": resourceschema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Capabilities reported by the agent.",
			},
			"edge_mtls_certificate_json": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Agent mTLS certificate metadata as JSON.",
			},
			"edge_security_mode": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Agent security mode.",
			},
			"edge_session_id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Agent session identifier.",
			},
		},
	}
}

func (r *EnvironmentResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type environmentModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	APIURL                  types.String `tfsdk:"api_url"`
	Enabled                 types.Bool   `tfsdk:"enabled"`
	Status                  types.String `tfsdk:"status"`
	EnrollmentToken         types.String `tfsdk:"enrollment_token"`
	EdgeAgentInstance       types.String `tfsdk:"edge_agent_instance"`
	EdgeCapabilities        types.List   `tfsdk:"edge_capabilities"`
	EdgeMTLSCertificateJSON types.String `tfsdk:"edge_mtls_certificate_json"`
	EdgeSecurityMode        types.String `tfsdk:"edge_security_mode"`
	EdgeSessionID           types.String `tfsdk:"edge_session_id"`
}

func (r *EnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan environmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.EnvironmentCreateRequest{}
	if isSetString(plan.Name) {
		v := plan.Name.ValueString()
		body.Name = &v
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		body.Enabled = &v
	}

	env, err := r.client.CreateEnvironment(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("create environment failed", err.Error())
		return
	}

	// Mint the single-use enrollment token the agent needs to join.
	snippet, err := r.client.GetDeploymentSnippet(ctx, env.ID)
	if err != nil {
		// The environment exists: keep it in state so it is not leaked.
		resp.Diagnostics.AddWarning("create environment enrollment token failed",
			"The environment was created but Arcane did not return an enrollment token: "+err.Error())
	}

	state := plan
	state.ID = types.StringValue(env.ID)
	state.EnrollmentToken = types.StringNull()
	if snippet != nil && snippet.EnrollmentToken != "" {
		state.EnrollmentToken = types.StringValue(snippet.EnrollmentToken)
	}
	applyEnvironment(ctx, &state, env, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state environmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	env, err := r.client.GetEnvironment(ctx, state.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read environment failed", err.Error())
		return
	}

	if !state.Name.IsNull() && !state.Name.IsUnknown() {
		state.Name = types.StringValue(env.Name)
	}
	if !state.Enabled.IsNull() && !state.Enabled.IsUnknown() {
		state.Enabled = types.BoolValue(env.Enabled)
	}
	applyEnvironment(ctx, &state, env, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state environmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.EnvironmentUpdateRequest{}
	if isSetString(plan.Name) {
		v := plan.Name.ValueString()
		body.Name = &v
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		body.Enabled = &v
	}

	env, err := r.client.UpdateEnvironment(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("update environment failed", err.Error())
		return
	}

	newState := plan
	newState.ID = state.ID
	newState.EnrollmentToken = state.EnrollmentToken
	applyEnvironment(ctx, &newState, env, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *EnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state environmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteEnvironment(ctx, state.ID.ValueString()); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete environment failed", err.Error())
	}
}

func (r *EnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyEnvironment copies the server-owned fields into state. desired carries
// the configured values: a configured name/enabled is echoed from the API and
// an unset one stays null.
func applyEnvironment(ctx context.Context, state *environmentModel, env *sdkclient.Environment, desired environmentModel) {
	state.Status = types.StringValue(env.Status)
	if isSetString(desired.Name) {
		state.Name = types.StringValue(env.Name)
	}
	if !desired.Enabled.IsNull() && !desired.Enabled.IsUnknown() {
		state.Enabled = types.BoolValue(env.Enabled)
	}
	state.APIURL = types.StringValue(env.APIURL)
	state.EdgeAgentInstance = nullableString(env.AgentInstance)
	state.EdgeCapabilities = stringsToList(ctx, env.AgentCapabilities)
	if env.AgentMTLSCertificate != nil {
		state.EdgeMTLSCertificateJSON = types.StringValue(mustJSON(env.AgentMTLSCertificate))
	} else {
		state.EdgeMTLSCertificateJSON = types.StringNull()
	}
	state.EdgeSecurityMode = nullableString(env.AgentSecurityMode)
	state.EdgeSessionID = nullableString(env.AgentSessionID)
	if state.EnrollmentToken.IsUnknown() {
		state.EnrollmentToken = types.StringNull()
	}
}
