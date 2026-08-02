package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ProjectResource{}
var _ resource.ResourceWithImportState = &ProjectResource{}
var _ resource.ResourceWithModifyPlan = &ProjectResource{}

type ProjectResource struct{ client *sdkclient.Client }

func NewProjectResource() resource.Resource { return &ProjectResource{} }

func (r *ProjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:      true,
				Description:   "Project ID",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id":  resourceschema.StringAttribute{Required: true, Description: "Environment ID", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"name":            resourceschema.StringAttribute{Required: true, Description: "Project name"},
			"compose_content": resourceschema.StringAttribute{Required: true, Description: "docker-compose.yml content"},
			"env_content":     resourceschema.StringAttribute{Optional: true, Description: ".env content"},
			"running":         resourceschema.BoolAttribute{Optional: true, Description: "If true, ensure project is running (compose up); if false, compose down. If unset, no lifecycle management."},
			"archived":        resourceschema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the project is archived.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}},
			// No Default here: the value is derived from redeploy_trigger in
			// planRedeploy when the practitioner did not set it. A static default
			// would fight that mirror on every plan (the default is applied before
			// ModifyPlan runs), leaving the resource with a perpetual diff.
			"redeploy_on_update": resourceschema.BoolAttribute{
				Optional:           true,
				Computed:           true,
				Description:        "Deprecated, use redeploy_trigger. Redeploy the project after updating compose/env content.",
				DeprecationMessage: "redeploy_on_update is deprecated, use redeploy_trigger instead: true is equivalent to redeploy_trigger = \"default\", false to redeploy_trigger = \"never\".",
			},
			"redeploy_trigger": resourceschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: redeployTriggerDescription,
				Validators: []validator.String{
					stringvalidator.OneOf(redeployTriggerValues...),
					stringvalidator.ConflictsWith(path.MatchRoot("redeploy_on_update")),
				},
			},
			"pull_on_update":      resourceschema.BoolAttribute{Optional: true, Computed: true, Description: "Pull images before redeploy.", Default: booldefault.StaticBool(false)},
			"remove_orphans":      resourceschema.BoolAttribute{Optional: true, Description: "When deploying (compose up), remove containers for services not defined in the compose file."},
			"fail_if_name_exists": resourceschema.BoolAttribute{Optional: true, Description: "If true, fail during the plan phase when a project with the same name already exists in the environment (including folders Arcane has discovered on disk), instead of letting Arcane auto-rename the new project with a numeric suffix. Defaults to false."},

			// Computed fields
			"path":              resourceschema.StringAttribute{Computed: true},
			"status":            resourceschema.StringAttribute{Computed: true},
			"service_count":     resourceschema.Int64Attribute{Computed: true},
			"running_count":     resourceschema.Int64Attribute{Computed: true},
			"created_at":        resourceschema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"updated_at":        resourceschema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"archived_at":       resourceschema.StringAttribute{Computed: true},
			"is_discovered":     resourceschema.BoolAttribute{Computed: true},
			"redeploy_disabled": resourceschema.BoolAttribute{Computed: true},
			"last_redeploy": resourceschema.StringAttribute{
				Computed:      true,
				Description:   lastRedeployDescription,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			// Delete options
			"remove_files":   resourceschema.BoolAttribute{Optional: true, Description: "Remove files on destroy"},
			"remove_volumes": resourceschema.BoolAttribute{Optional: true, Description: "Remove volumes on destroy"},
		},
	}
}

func (r *ProjectResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

// projectServerComputed lists the attributes Update rewrites from the API
// response. Keep it in sync with the assignments at the end of Update; see
// planServerComputedUnknown for what goes wrong when an attribute is missing.
var projectServerComputed = []serverComputedAttr{
	{"path", types.StringUnknown()},
	{"status", types.StringUnknown()},
	{"service_count", types.Int64Unknown()},
	{"running_count", types.Int64Unknown()},
	{"archived_at", types.StringUnknown()},
	{"is_discovered", types.BoolUnknown()},
	{"redeploy_disabled", types.BoolUnknown()},
}

// ModifyPlan resolves the effective redeploy_trigger and enforces the optional
// fail_if_name_exists check during the plan phase.
func (r *ProjectResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Destroy plan: nothing to modify.
		return
	}

	r.planRedeploy(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	r.planFailIfNameExists(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	// Last: it keys off the plan the steps above produced.
	planServerComputedUnknown(ctx, req, resp, projectServerComputed)
}

// planRedeploy writes the effective redeploy trigger into the plan and, when
// that trigger means the next apply will redeploy, marks last_redeploy unknown.
//
// Marking last_redeploy unknown is what makes redeploy_trigger = "always" work:
// Terraform only calls Update when the plan differs from state, so without an
// attribute that changes there is nothing to hang a redeploy on. The unknown is
// only injected when a redeploy is actually expected, so the other triggers keep
// producing empty plans when nothing changed.
func (r *ProjectResource) planRedeploy(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var configured types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("redeploy_trigger"), &configured)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if configured.IsUnknown() {
		// The configured value only becomes known at apply time, and a planned
		// value may not contradict the configuration. Update resolves it then.
		return
	}

	trigger, diags := resolveRedeployTrigger(ctx, req.Config, "redeploy_on_update")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("redeploy_trigger"), types.StringValue(trigger))...)

	// Keep the deprecated mirror in sync when the practitioner did not set it, so
	// state never claims redeploy_on_update = true while the trigger says never.
	var legacy types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("redeploy_on_update"), &legacy)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if legacy.IsNull() {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("redeploy_on_update"), types.BoolValue(trigger != redeployTriggerNever))...)
	}

	if req.State.Raw.IsNull() {
		// Create: Create() seeds last_redeploy itself.
		return
	}

	var plan, state projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	contentChanged := !plan.Compose.Equal(state.Compose) || !plan.Env.Equal(state.Env)
	if !redeployPlanned(trigger, contentChanged, !req.Plan.Raw.Equal(req.State.Raw)) {
		return
	}
	if !projectRedeployAllowed(plan) {
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("last_redeploy"), types.StringUnknown())...)
}

// planFailIfNameExists enforces the optional fail_if_name_exists check during the
// plan phase. When enabled, creating a project whose name already exists in the
// environment (a registered project or a folder Arcane has discovered on disk)
// fails the plan instead of letting Arcane silently auto-rename the project with
// a numeric suffix, which would make the resource non-deterministic.
func (r *ProjectResource) planFailIfNameExists(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}
	if !req.State.Raw.IsNull() {
		return
	}

	var plan projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.FailIfNameExists.ValueBool() {
		return
	}
	if plan.Name.IsUnknown() || plan.Name.IsNull() || plan.EnvironmentID.IsUnknown() || plan.EnvironmentID.IsNull() {
		return
	}

	name := plan.Name.ValueString()
	envID := plan.EnvironmentID.ValueString()
	existing, err := r.client.ListProjects(ctx, envID)
	if err != nil {
		resp.Diagnostics.AddError("failed to check for existing project name", err.Error())
		return
	}
	for _, p := range existing {
		if p.Name == name {
			resp.Diagnostics.AddError(
				"project name already exists",
				fmt.Sprintf("A project named %q already exists in environment %q (id %q). Arcane would auto-rename the new project (e.g. %q-1), which is non-deterministic. Choose a unique name, import the existing project, or set fail_if_name_exists = false to allow the rename.", name, envID, p.ID, name),
			)
			return
		}
	}
}

type projectModel struct {
	ID               types.String `tfsdk:"id"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	Name             types.String `tfsdk:"name"`
	Compose          types.String `tfsdk:"compose_content"`
	Env              types.String `tfsdk:"env_content"`
	Running          types.Bool   `tfsdk:"running"`
	Archived         types.Bool   `tfsdk:"archived"`
	RedeployOnUpdate types.Bool   `tfsdk:"redeploy_on_update"`
	RedeployTrigger  types.String `tfsdk:"redeploy_trigger"`
	LastRedeploy     types.String `tfsdk:"last_redeploy"`
	PullOnUpdate     types.Bool   `tfsdk:"pull_on_update"`
	RemoveOrphans    types.Bool   `tfsdk:"remove_orphans"`
	FailIfNameExists types.Bool   `tfsdk:"fail_if_name_exists"`
	Path             types.String `tfsdk:"path"`
	Status           types.String `tfsdk:"status"`
	ServiceCount     types.Int64  `tfsdk:"service_count"`
	RunningCount     types.Int64  `tfsdk:"running_count"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
	ArchivedAt       types.String `tfsdk:"archived_at"`
	IsDiscovered     types.Bool   `tfsdk:"is_discovered"`
	RedeployDisabled types.Bool   `tfsdk:"redeploy_disabled"`
	RemoveFiles      types.Bool   `tfsdk:"remove_files"`
	RemoveVolumes    types.Bool   `tfsdk:"remove_volumes"`
}

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if boolValue(plan.Archived) && boolValue(plan.Running) {
		resp.Diagnostics.AddError("invalid project lifecycle", "archived cannot be true when running is true")
		return
	}

	redeployTrigger, redeployOnUpdate, diags := resolvedRedeployAttrs(ctx, req.Config, "redeploy_on_update", plan.RedeployTrigger, plan.RedeployOnUpdate)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.ProjectCreateRequest{Name: plan.Name.ValueString(), ComposeContent: plan.Compose.ValueString()}
	if !plan.Env.IsNull() && !plan.Env.IsUnknown() {
		v := plan.Env.ValueString()
		body.EnvContent = &v
	}

	envID := plan.EnvironmentID.ValueString()
	out, err := r.client.CreateProject(ctx, envID, body)
	if err != nil {
		resp.Diagnostics.AddError("create project failed", err.Error())
		return
	}

	// Manage lifecycle if requested
	if !plan.Running.IsNull() && !plan.Running.IsUnknown() {
		if plan.Running.ValueBool() {
			if err := r.client.UpProject(ctx, envID, out.ID, projectDeployOpts(plan)); err != nil {
				resp.Diagnostics.AddError("project up failed", err.Error())
				return
			}
		} else {
			if err := r.client.DownProject(ctx, envID, out.ID); err != nil {
				resp.Diagnostics.AddError("project down failed", err.Error())
				return
			}
		}
		if det, derr := r.client.GetProject(ctx, envID, out.ID); derr == nil {
			out.Status = det.Status
			out.RunningCount = det.RunningCount
			out.ServiceCount = det.ServiceCount
			out.UpdatedAt = det.UpdatedAt
		}
	}

	if boolValue(plan.Archived) {
		if err := r.client.DownProject(ctx, envID, out.ID); err != nil {
			resp.Diagnostics.AddError("project down before archive failed", err.Error())
			return
		}
		if err := r.client.ArchiveProject(ctx, envID, out.ID); err != nil {
			resp.Diagnostics.AddError("project archive failed", err.Error())
			return
		}
		if det, derr := r.client.GetProject(ctx, envID, out.ID); derr == nil {
			out.Status = det.Status
			out.RunningCount = det.RunningCount
			out.ServiceCount = det.ServiceCount
			out.IsArchived = det.IsArchived
			out.ArchivedAt = det.ArchivedAt
			out.UpdatedAt = det.UpdatedAt
		}
	}

	state := projectModel{
		ID:               types.StringValue(out.ID),
		EnvironmentID:    plan.EnvironmentID,
		Name:             types.StringValue(out.Name),
		Compose:          plan.Compose,
		Env:              plan.Env,
		Path:             types.StringValue(out.Path),
		Status:           types.StringValue(out.Status),
		ServiceCount:     types.Int64Value(int64(out.ServiceCount)),
		RunningCount:     types.Int64Value(int64(out.RunningCount)),
		CreatedAt:        types.StringValue(out.CreatedAt),
		UpdatedAt:        types.StringValue(out.UpdatedAt),
		Archived:         types.BoolValue(out.IsArchived),
		ArchivedAt:       nullableString(out.ArchivedAt),
		RemoveFiles:      plan.RemoveFiles,
		RemoveVolumes:    plan.RemoveVolumes,
		Running:          plan.Running,
		RedeployOnUpdate: redeployOnUpdate,
		RedeployTrigger:  redeployTrigger,
		// Create deploys through "up", not "redeploy": nothing has been
		// redeployed yet.
		LastRedeploy:     types.StringNull(),
		PullOnUpdate:     plan.PullOnUpdate,
		RemoveOrphans:    plan.RemoveOrphans,
		FailIfNameExists: plan.FailIfNameExists,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := state.EnvironmentID.ValueString()
	projID := state.ID.ValueString()

	out, err := r.client.GetProject(ctx, envID, projID)
	if err != nil {
		if r.client.IsResourceGone(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read project failed", err.Error())
		return
	}

	state.Name = types.StringValue(out.Name)
	state.Path = types.StringValue(out.Path)
	state.Status = types.StringValue(out.Status)
	state.ServiceCount = types.Int64Value(int64(out.ServiceCount))
	state.RunningCount = types.Int64Value(int64(out.RunningCount))
	state.Archived = types.BoolValue(out.IsArchived)
	state.ArchivedAt = nullableString(out.ArchivedAt)
	state.IsDiscovered = types.BoolValue(out.IsDiscovered)
	state.RedeployDisabled = types.BoolValue(out.RedeployDisabled)
	// Leave created_at and updated_at unchanged to avoid plan inconsistency on server-side timestamp changes
	if out.ComposeContent != nil {
		state.Compose = types.StringValue(*out.ComposeContent)
	}
	if !state.Env.IsNull() && !state.Env.IsUnknown() && out.EnvContent != nil {
		state.Env = types.StringValue(*out.EnvContent)
	}
	// Preserve configuration values that have defaults
	// PullOnUpdate, RedeployOnUpdate, Running, RemoveFiles, RemoveVolumes are already in state

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectModel
	var state projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if boolValue(plan.Archived) && boolValue(plan.Running) {
		resp.Diagnostics.AddError("invalid project lifecycle", "archived cannot be true when running is true")
		return
	}

	redeployTrigger, redeployOnUpdate, diags := resolvedRedeployAttrs(ctx, req.Config, "redeploy_on_update", plan.RedeployTrigger, plan.RedeployOnUpdate)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	envID := state.EnvironmentID.ValueString()
	projID := state.ID.ValueString()
	// Compare against the prior state before it gets overwritten below: the
	// request body always carries the compose/env content, so it cannot tell us
	// whether the content actually changed.
	contentChanged := !plan.Compose.Equal(state.Compose) || !plan.Env.Equal(state.Env)
	body := sdkclient.ProjectUpdateRequest{}
	if !plan.Compose.IsNull() && !plan.Compose.IsUnknown() {
		v := plan.Compose.ValueString()
		body.ComposeContent = &v
	}
	if !plan.Env.IsNull() && !plan.Env.IsUnknown() {
		v := plan.Env.ValueString()
		body.EnvContent = &v
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		v := plan.Name.ValueString()
		body.Name = &v
	}

	out, err := r.client.UpdateProject(ctx, envID, projID, body)
	if err != nil {
		resp.Diagnostics.AddError("update project failed", err.Error())
		return
	}

	// Redeploy according to redeploy_trigger, as long as the project is meant to
	// be up. The trigger is resolved during plan (see planRedeploy), so the plan
	// value is authoritative here.
	redeployed := false
	if shouldRedeploy(redeployTrigger.ValueString(), contentChanged) && projectRedeployAllowed(plan) {
		// Optionally pull first
		if boolValue(plan.PullOnUpdate) {
			if err := r.client.PullProjectImages(ctx, envID, projID); err != nil {
				resp.Diagnostics.AddError("project image pull failed", err.Error())
				return
			}
		}

		if err := r.client.RedeployProject(ctx, envID, projID); err != nil {
			resp.Diagnostics.AddError("project redeploy failed", err.Error())
			return
		}
		redeployed = true
		if det, derr := r.client.GetProject(ctx, envID, projID); derr == nil {
			out.Status = det.Status
		}
	}

	if !plan.Archived.IsNull() && !plan.Archived.IsUnknown() {
		desiredArchived := plan.Archived.ValueBool()
		currentArchived := false
		if !state.Archived.IsNull() && !state.Archived.IsUnknown() {
			currentArchived = state.Archived.ValueBool()
		}
		if desiredArchived != currentArchived {
			if desiredArchived {
				if err := r.client.DownProject(ctx, envID, projID); err != nil {
					resp.Diagnostics.AddError("project down before archive failed", err.Error())
					return
				}
				if err := r.client.ArchiveProject(ctx, envID, projID); err != nil {
					resp.Diagnostics.AddError("project archive failed", err.Error())
					return
				}
			} else {
				if err := r.client.UnarchiveProject(ctx, envID, projID); err != nil {
					resp.Diagnostics.AddError("project unarchive failed", err.Error())
					return
				}
			}
			if det, derr := r.client.GetProject(ctx, envID, projID); derr == nil {
				out.Status = det.Status
				out.ServiceCount = det.ServiceCount
				out.RunningCount = det.RunningCount
				out.IsArchived = det.IsArchived
				out.ArchivedAt = det.ArchivedAt
				out.IsDiscovered = det.IsDiscovered
				out.RedeployDisabled = det.RedeployDisabled
			}
		}
	}

	// Lifecycle manage if configured
	if !boolValue(plan.Archived) && !plan.Running.IsNull() && !plan.Running.IsUnknown() {
		desired := plan.Running.ValueBool()
		current := state.Running.ValueBool()
		if desired != current {
			if desired {
				if err := r.client.UpProject(ctx, envID, projID, projectDeployOpts(plan)); err != nil {
					resp.Diagnostics.AddError("project up failed", err.Error())
					return
				}
			} else {
				if err := r.client.DownProject(ctx, envID, projID); err != nil {
					resp.Diagnostics.AddError("project down failed", err.Error())
					return
				}
			}
			if det, derr := r.client.GetProject(ctx, envID, projID); derr == nil {
				out.Status = det.Status
			}
			state.Running = plan.Running
		}
	}

	state.Name = types.StringValue(out.Name)
	state.Path = types.StringValue(out.Path)
	state.Status = types.StringValue(out.Status)
	state.ServiceCount = types.Int64Value(int64(out.ServiceCount))
	state.RunningCount = types.Int64Value(int64(out.RunningCount))
	if !plan.Archived.IsNull() && !plan.Archived.IsUnknown() {
		state.Archived = types.BoolValue(out.IsArchived)
	} else {
		state.Archived = plan.Archived
	}
	state.ArchivedAt = nullableString(out.ArchivedAt)
	state.IsDiscovered = types.BoolValue(out.IsDiscovered)
	state.RedeployDisabled = types.BoolValue(out.RedeployDisabled)
	// Leave created_at and updated_at unchanged to avoid plan inconsistency
	state.Compose = plan.Compose
	state.Env = plan.Env
	state.PullOnUpdate = plan.PullOnUpdate
	state.RedeployOnUpdate = redeployOnUpdate
	state.RedeployTrigger = redeployTrigger
	// last_redeploy only moves when the plan left it unknown (planRedeploy marks
	// it whenever a redeploy is expected); otherwise the planned value, which
	// UseStateForUnknown pinned to the prior state, has to be written back
	// verbatim or the apply is inconsistent with the plan.
	if plan.LastRedeploy.IsUnknown() {
		if redeployed {
			state.LastRedeploy = types.StringValue(redeployTimestamp())
		}
	} else {
		state.LastRedeploy = plan.LastRedeploy
	}
	state.RemoveOrphans = plan.RemoveOrphans
	state.FailIfNameExists = plan.FailIfNameExists
	state.RemoveFiles = plan.RemoveFiles
	state.RemoveVolumes = plan.RemoveVolumes
	// Persist running unconditionally: the lifecycle block above only assigns it
	// when desired != current, so other transitions would otherwise keep the
	// stale prior value and produce an inconsistent-result error.
	state.Running = plan.Running
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := state.EnvironmentID.ValueString()
	projID := state.ID.ValueString()
	opts := sdkclient.ProjectDestroyOptions{RemoveFiles: state.RemoveFiles.ValueBool(), RemoveVolumes: state.RemoveVolumes.ValueBool()}
	if err := r.client.DestroyProject(ctx, envID, projID, opts); err != nil {
		if r.client.IsResourceGone(err) {
			return
		}
		resp.Diagnostics.AddError("destroy project failed", err.Error())
	}
}

// projectRedeployAllowed reports whether redeploying matches the desired state.
// Redeploying brings a project up, so it must not run against a project that is
// archived or explicitly configured as not running.
func projectRedeployAllowed(m projectModel) bool {
	if boolValue(m.Archived) {
		return false
	}
	if !m.Running.IsNull() && !m.Running.IsUnknown() {
		return m.Running.ValueBool()
	}
	return true
}

// projectDeployOpts builds the optional deploy body for the "up" endpoint from
// the model. Returns nil when no deploy option is configured so the request
// body is omitted.
func projectDeployOpts(m projectModel) *sdkclient.ProjectDeployOptions {
	if m.RemoveOrphans.IsNull() || m.RemoveOrphans.IsUnknown() {
		return nil
	}
	v := m.RemoveOrphans.ValueBool()
	return &sdkclient.ProjectDeployOptions{RemoveOrphans: &v}
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by envID:projectID
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("invalid import id", "expected env_id:project_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
