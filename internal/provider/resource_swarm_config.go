package provider

import (
	"context"
	"strings"
	"unicode/utf8"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SwarmConfigResource{}
var _ resource.ResourceWithImportState = &SwarmConfigResource{}

type SwarmConfigResource struct{ client *sdkclient.Client }

func NewSwarmConfigResource() resource.Resource { return &SwarmConfigResource{} }

func (r *SwarmConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_swarm_config"
}

func (r *SwarmConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manages a Docker Swarm config in an Arcane environment.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:      true,
				Description:   "Swarm config ID",
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
				Description: "Config name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"data": resourceschema.StringAttribute{
				Required:    true,
				Description: "Config content (plaintext). The provider encodes this to base64 for the API.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"labels": resourceschema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Config labels",
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

func (r *SwarmConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type swarmConfigModel struct {
	ID            types.String `tfsdk:"id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	Name          types.String `tfsdk:"name"`
	Data          types.String `tfsdk:"data"`
	Labels        types.Map    `tfsdk:"labels"`
	VersionIndex  types.Int64  `tfsdk:"version_index"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *SwarmConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan swarmConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	spec := sdkclient.DockerSwarmConfigSpec{
		Name: plan.Name.ValueString(),
		Data: sdkclient.EncodeSwarmConfigData(plan.Data.ValueString()),
	}
	if !plan.Labels.IsNull() && !plan.Labels.IsUnknown() {
		spec.Labels = mapFromStringMap(ctx, plan.Labels)
	}

	config, err := r.client.CreateSwarmConfig(ctx, plan.EnvironmentID.ValueString(), sdkclient.SwarmConfigCreateRequest{Spec: spec})
	if err != nil {
		resp.Diagnostics.AddError("create swarm config failed", err.Error())
		return
	}

	state := swarmConfigModel{
		ID:            types.StringValue(config.ID),
		EnvironmentID: plan.EnvironmentID,
		Name:          types.StringValue(config.Spec.Name),
		Data:          plan.Data,
		Labels:        stringMapToMap(ctx, config.Spec.Labels),
		VersionIndex:  types.Int64Value(config.Version.Index),
		CreatedAt:     types.StringValue(config.CreatedAt),
		UpdatedAt:     types.StringValue(config.UpdatedAt),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SwarmConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state swarmConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetSwarmConfig(ctx, state.EnvironmentID.ValueString(), state.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read swarm config failed", err.Error())
		return
	}

	// A null `data` means this state came from ImportState: `data` is Required,
	// so a config Terraform created or refreshed always carries it. Everything
	// under `imported` exists to fill the gaps ImportState leaves, so that the
	// first plan after an import does not read as a destroy/recreate.
	imported := state.Data.IsNull() || state.Data.IsUnknown()

	state.Name = types.StringValue(config.Spec.Name)
	// Outside an import, a null labels map means "unconfigured" and must stay
	// null: adopting server labels there would force a spurious replace.
	if imported || (!state.Labels.IsNull() && !state.Labels.IsUnknown()) {
		state.Labels = stringMapToMap(ctx, config.Spec.Labels)
	}
	state.VersionIndex = types.Int64Value(config.Version.Index)
	state.CreatedAt = types.StringValue(config.CreatedAt)
	state.UpdatedAt = types.StringValue(config.UpdatedAt)

	// Keep the configured plaintext when we already have it: the API hands back
	// exactly the base64 we sent, so re-decoding a known value buys nothing.
	// Swarm secrets cannot do this at all — their API never returns spec.Data.
	if imported {
		if plaintext, ok := decodeSwarmConfigData(config.Spec.Data); ok {
			state.Data = types.StringValue(plaintext)
		} else {
			resp.Diagnostics.AddWarning(
				"swarm config content unavailable",
				"The API did not return decodable UTF-8 content for swarm config "+state.ID.ValueString()+
					". Set `data` in the configuration to match the existing config, otherwise the next apply replaces it.",
			)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SwarmConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("update not supported", "Swarm configs are immutable and must be replaced when data, name, or labels change.")
}

func (r *SwarmConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state swarmConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSwarmConfig(ctx, state.EnvironmentID.ValueString(), state.ID.ValueString()); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete swarm config failed", err.Error())
	}
}

// decodeSwarmConfigData turns the base64 the API returns back into the
// plaintext held in state. Content that is not valid UTF-8 (an archive, a
// binary keystore) has no Terraform string representation, so it reports false
// rather than writing corrupt state.
func decodeSwarmConfigData(encoded string) (string, bool) {
	if encoded == "" {
		return "", false
	}
	plaintext, err := sdkclient.DecodeSwarmConfigData(encoded)
	if err != nil || !utf8.ValidString(plaintext) {
		return "", false
	}
	return plaintext, true
}

func (r *SwarmConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("invalid import id", "expected env_id:config_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
