package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SwarmStackResource{}
var _ resource.ResourceWithImportState = &SwarmStackResource{}

type SwarmStackResource struct{ client *sdkclient.Client }

func NewSwarmStackResource() resource.Resource { return &SwarmStackResource{} }

func (r *SwarmStackResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_swarm_stack"
}

func (r *SwarmStackResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manages a Docker Swarm stack in an Arcane environment.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:      true,
				Description:   "Stack name (resource ID)",
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
				Description: "Swarm stack name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"compose_content": resourceschema.StringAttribute{
				Required:    true,
				Description: "Docker Compose content for stack deployment",
			},
			"env_content": resourceschema.StringAttribute{
				Optional:    true,
				Description: ".env content for stack deployment",
			},
			"prune": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Prune services that are no longer referenced in the compose file during deploy",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"resolve_image": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Image resolution mode for deploy (for example always, changed, or never)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"with_registry_auth": resourceschema.BoolAttribute{
				Optional:    true,
				Description: "Forward registry authentication to swarm agents during deploy",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"working_dir": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Working directory used by Arcane for compose include resolution",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"namespace": resourceschema.StringAttribute{Computed: true, Description: "Docker namespace for the stack"},
			"services":  resourceschema.Int64Attribute{Computed: true, Description: "Number of services in the stack"},
			"created_at": resourceschema.StringAttribute{
				Computed:      true,
				Description:   "Creation timestamp",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": resourceschema.StringAttribute{
				Computed:      true,
				Description:   "Last update timestamp",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *SwarmStackResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type swarmStackModel struct {
	ID               types.String `tfsdk:"id"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	Name             types.String `tfsdk:"name"`
	ComposeContent   types.String `tfsdk:"compose_content"`
	EnvContent       types.String `tfsdk:"env_content"`
	Prune            types.Bool   `tfsdk:"prune"`
	ResolveImage     types.String `tfsdk:"resolve_image"`
	WithRegistryAuth types.Bool   `tfsdk:"with_registry_auth"`
	WorkingDir       types.String `tfsdk:"working_dir"`
	Namespace        types.String `tfsdk:"namespace"`
	Services         types.Int64  `tfsdk:"services"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (r *SwarmStackResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan swarmStackModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.SwarmStackDeployRequest{
		Name:           plan.Name.ValueString(),
		ComposeContent: plan.ComposeContent.ValueString(),
	}
	if !plan.EnvContent.IsNull() && !plan.EnvContent.IsUnknown() {
		v := plan.EnvContent.ValueString()
		body.EnvContent = &v
	}
	if !plan.Prune.IsNull() && !plan.Prune.IsUnknown() {
		v := plan.Prune.ValueBool()
		body.Prune = &v
	}
	if !plan.ResolveImage.IsNull() && !plan.ResolveImage.IsUnknown() {
		v := plan.ResolveImage.ValueString()
		body.ResolveImage = &v
	}
	if !plan.WithRegistryAuth.IsNull() && !plan.WithRegistryAuth.IsUnknown() {
		v := plan.WithRegistryAuth.ValueBool()
		body.WithRegistryAuth = &v
	}
	if !plan.WorkingDir.IsNull() && !plan.WorkingDir.IsUnknown() {
		v := plan.WorkingDir.ValueString()
		body.WorkingDir = &v
	}

	envID := plan.EnvironmentID.ValueString()
	out, err := r.client.DeploySwarmStack(ctx, envID, body)
	if err != nil {
		resp.Diagnostics.AddError("deploy swarm stack failed", err.Error())
		return
	}

	inspect, err := r.client.GetSwarmStack(ctx, envID, out.Name)
	if err != nil {
		resp.Diagnostics.AddError("read swarm stack after deploy failed", err.Error())
		return
	}

	state := swarmStackModel{
		ID:               types.StringValue(out.Name),
		EnvironmentID:    plan.EnvironmentID,
		Name:             types.StringValue(out.Name),
		ComposeContent:   plan.ComposeContent,
		EnvContent:       plan.EnvContent,
		Prune:            plan.Prune,
		ResolveImage:     plan.ResolveImage,
		WithRegistryAuth: plan.WithRegistryAuth,
		WorkingDir:       plan.WorkingDir,
		Namespace:        types.StringValue(inspect.Namespace),
		Services:         types.Int64Value(inspect.Services),
		CreatedAt:        types.StringValue(inspect.CreatedAt),
		UpdatedAt:        types.StringValue(inspect.UpdatedAt),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SwarmStackResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state swarmStackModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envID := state.EnvironmentID.ValueString()
	stackName := state.ID.ValueString()

	inspect, err := r.client.GetSwarmStack(ctx, envID, stackName)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read swarm stack failed", err.Error())
		return
	}

	source, err := r.client.GetSwarmStackSource(ctx, envID, stackName)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read swarm stack source failed", err.Error())
		return
	}

	state.ID = types.StringValue(inspect.Name)
	state.Name = types.StringValue(inspect.Name)
	state.ComposeContent = types.StringValue(source.ComposeContent)
	if source.EnvContent != "" {
		state.EnvContent = types.StringValue(source.EnvContent)
	} else {
		state.EnvContent = types.StringNull()
	}
	state.Namespace = types.StringValue(inspect.Namespace)
	state.Services = types.Int64Value(inspect.Services)
	state.CreatedAt = types.StringValue(inspect.CreatedAt)
	state.UpdatedAt = types.StringValue(inspect.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SwarmStackResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan swarmStackModel
	var state swarmStackModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envID := state.EnvironmentID.ValueString()
	stackName := state.ID.ValueString()
	body := sdkclient.SwarmStackSourceUpdateRequest{ComposeContent: plan.ComposeContent.ValueString()}
	if !plan.EnvContent.IsNull() && !plan.EnvContent.IsUnknown() {
		v := plan.EnvContent.ValueString()
		body.EnvContent = &v
	}

	source, err := r.client.UpdateSwarmStackSource(ctx, envID, stackName, body)
	if err != nil {
		resp.Diagnostics.AddError("update swarm stack source failed", err.Error())
		return
	}

	inspect, err := r.client.GetSwarmStack(ctx, envID, stackName)
	if err != nil {
		resp.Diagnostics.AddError("read swarm stack after update failed", err.Error())
		return
	}

	state.Name = types.StringValue(inspect.Name)
	state.ComposeContent = types.StringValue(source.ComposeContent)
	if source.EnvContent != "" {
		state.EnvContent = types.StringValue(source.EnvContent)
	} else {
		state.EnvContent = types.StringNull()
	}
	state.Namespace = types.StringValue(inspect.Namespace)
	state.Services = types.Int64Value(inspect.Services)
	state.UpdatedAt = types.StringValue(inspect.UpdatedAt)
	state.Prune = plan.Prune
	state.ResolveImage = plan.ResolveImage
	state.WithRegistryAuth = plan.WithRegistryAuth
	state.WorkingDir = plan.WorkingDir

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SwarmStackResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state swarmStackModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSwarmStack(ctx, state.EnvironmentID.ValueString(), state.ID.ValueString()); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete swarm stack failed", err.Error())
	}
}

func (r *SwarmStackResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("invalid import id", "expected env_id:stack_name")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}
