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
				Required:    true,
				Description: "Agent API URL (e.g., http://host:agent-port)",
			},
			"access_token": resourceschema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Access token for agent pairing (optional)",
			},
			"bootstrap_token": resourceschema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Bootstrap token for remote agent pairing (optional)",
			},
			"use_api_key": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "When true, generates an API key for agent pairing.",
			},
			"is_edge": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Whether the environment uses edge transport.",
			},
			"enabled": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Whether the environment is enabled.",
			},
			"regenerate_api_key": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Regenerate the pairing API key on update.",
			},

			// Computed fields
			"status": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Environment status.",
			},
			"api_key": resourceschema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Pairing API key (only returned on creation).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"edge_agent_instance": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Edge agent instance identifier.",
			},
			"edge_capabilities": resourceschema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Capabilities reported by the edge agent.",
			},
			"edge_mtls_certificate_json": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Edge mTLS certificate metadata as JSON.",
			},
			"edge_security_mode": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Edge security mode.",
			},
			"edge_session_id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Edge session identifier.",
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
	AccessToken             types.String `tfsdk:"access_token"`
	BootstrapToken          types.String `tfsdk:"bootstrap_token"`
	UseAPIKey               types.Bool   `tfsdk:"use_api_key"`
	IsEdge                  types.Bool   `tfsdk:"is_edge"`
	Enabled                 types.Bool   `tfsdk:"enabled"`
	RegenerateAPIKey        types.Bool   `tfsdk:"regenerate_api_key"`
	Status                  types.String `tfsdk:"status"`
	APIKey                  types.String `tfsdk:"api_key"`
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

	body := sdkclient.EnvironmentCreateRequest{
		APIURL: plan.APIURL.ValueString(),
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		v := plan.Name.ValueString()
		body.Name = &v
	}
	if !plan.AccessToken.IsNull() && !plan.AccessToken.IsUnknown() {
		v := plan.AccessToken.ValueString()
		body.AccessToken = &v
	}
	if !plan.BootstrapToken.IsNull() && !plan.BootstrapToken.IsUnknown() {
		v := plan.BootstrapToken.ValueString()
		body.BootstrapToken = &v
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		body.Enabled = &v
	}
	if !plan.IsEdge.IsNull() && !plan.IsEdge.IsUnknown() {
		v := plan.IsEdge.ValueBool()
		body.IsEdge = &v
	}
	if !plan.UseAPIKey.IsNull() && !plan.UseAPIKey.IsUnknown() {
		v := plan.UseAPIKey.ValueBool()
		body.UseAPIKey = &v
	}

	env, err := r.client.CreateEnvironment(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("create environment failed", err.Error())
		return
	}

	state := environmentModel{
		ID:               types.StringValue(env.ID),
		APIURL:           types.StringValue(env.APIURL),
		AccessToken:      plan.AccessToken,
		BootstrapToken:   plan.BootstrapToken,
		UseAPIKey:        plan.UseAPIKey,
		RegenerateAPIKey: plan.RegenerateAPIKey,
		Status:           types.StringValue(env.Status),
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = types.StringValue(env.Name)
	} else {
		state.Name = plan.Name
	}
	if !plan.IsEdge.IsNull() && !plan.IsEdge.IsUnknown() {
		state.IsEdge = types.BoolValue(env.IsEdge)
	} else {
		state.IsEdge = plan.IsEdge
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		state.Enabled = types.BoolValue(env.Enabled)
	} else {
		state.Enabled = plan.Enabled
	}
	if env.APIKey != "" {
		state.APIKey = types.StringValue(env.APIKey)
	}
	applyEnvironmentEdgeFields(ctx, &state, env)

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

	state.APIURL = types.StringValue(env.APIURL)
	state.Status = types.StringValue(env.Status)
	if !state.Name.IsNull() && !state.Name.IsUnknown() {
		state.Name = types.StringValue(env.Name)
	}
	if !state.Enabled.IsNull() && !state.Enabled.IsUnknown() {
		state.Enabled = types.BoolValue(env.Enabled)
	}
	if !state.IsEdge.IsNull() && !state.IsEdge.IsUnknown() {
		state.IsEdge = types.BoolValue(env.IsEdge)
	}
	applyEnvironmentEdgeFields(ctx, &state, env)
	// access_token/bootstrap_token/use_api_key remain as configured
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan environmentModel
	var state environmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.EnvironmentUpdateRequest{}
	if !plan.APIURL.IsNull() && !plan.APIURL.IsUnknown() {
		v := plan.APIURL.ValueString()
		body.APIURL = &v
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		v := plan.Name.ValueString()
		body.Name = &v
	}
	if !plan.AccessToken.IsNull() && !plan.AccessToken.IsUnknown() {
		v := plan.AccessToken.ValueString()
		body.AccessToken = &v
	}
	if !plan.BootstrapToken.IsNull() && !plan.BootstrapToken.IsUnknown() {
		v := plan.BootstrapToken.ValueString()
		body.BootstrapToken = &v
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		body.Enabled = &v
	}
	if !plan.RegenerateAPIKey.IsNull() && !plan.RegenerateAPIKey.IsUnknown() {
		v := plan.RegenerateAPIKey.ValueBool()
		body.RegenerateAPIKey = &v
	}

	env, err := r.client.UpdateEnvironment(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("update environment failed", err.Error())
		return
	}

	state.Name = types.StringValue(env.Name)
	state.APIURL = types.StringValue(env.APIURL)
	state.Status = types.StringValue(env.Status)
	state.Enabled = types.BoolValue(env.Enabled)
	state.IsEdge = types.BoolValue(env.IsEdge)
	applyEnvironmentEdgeFields(ctx, &state, env)
	if !plan.AccessToken.IsNull() && !plan.AccessToken.IsUnknown() {
		state.AccessToken = plan.AccessToken
	}
	if !plan.BootstrapToken.IsNull() && !plan.BootstrapToken.IsUnknown() {
		state.BootstrapToken = plan.BootstrapToken
	}
	if !plan.UseAPIKey.IsNull() && !plan.UseAPIKey.IsUnknown() {
		state.UseAPIKey = plan.UseAPIKey
	}
	if !plan.RegenerateAPIKey.IsNull() && !plan.RegenerateAPIKey.IsUnknown() {
		state.RegenerateAPIKey = plan.RegenerateAPIKey
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
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

func applyEnvironmentEdgeFields(ctx context.Context, state *environmentModel, env *sdkclient.Environment) {
	state.EdgeAgentInstance = nullableString(env.EdgeAgentInstance)
	state.EdgeCapabilities = stringsToList(ctx, env.EdgeCapabilities)
	if env.EdgeMTLSCertificate != nil {
		state.EdgeMTLSCertificateJSON = types.StringValue(mustJSON(env.EdgeMTLSCertificate))
	} else {
		state.EdgeMTLSCertificateJSON = types.StringNull()
	}
	state.EdgeSecurityMode = nullableString(env.EdgeSecurityMode)
	state.EdgeSessionID = nullableString(env.EdgeSessionID)
}
