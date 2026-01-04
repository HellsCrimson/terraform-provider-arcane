package provider

import (
    "context"
    "strings"

    "terraform-provider-arcane/internal/sdkclient"

    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ProjectResource{}
var _ resource.ResourceWithImportState = &ProjectResource{}

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
			"environment_id":     resourceschema.StringAttribute{Required: true, Description: "Environment ID"},
			"name":               resourceschema.StringAttribute{Required: true, Description: "Project name"},
			"compose_content":    resourceschema.StringAttribute{Required: true, Description: "docker-compose.yml content"},
			"env_content":        resourceschema.StringAttribute{Optional: true, Description: ".env content"},
			"running":            resourceschema.BoolAttribute{Optional: true, Description: "If true, ensure project is running (compose up); if false, compose down. If unset, no lifecycle management."},
			"redeploy_on_update": resourceschema.BoolAttribute{Optional: true, Computed: true, Description: "Redeploy the project after updating compose/env content.", Default: booldefault.StaticBool(true)},

			// Computed fields
			"path":          resourceschema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"status":        resourceschema.StringAttribute{Computed: true},
			"service_count": resourceschema.Int64Attribute{Computed: true},
			"running_count": resourceschema.Int64Attribute{Computed: true},
			"created_at":    resourceschema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"updated_at":    resourceschema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},

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

type projectModel struct {
	ID               types.String `tfsdk:"id"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	Name             types.String `tfsdk:"name"`
	Compose          types.String `tfsdk:"compose_content"`
	Env              types.String `tfsdk:"env_content"`
	Running          types.Bool   `tfsdk:"running"`
	RedeployOnUpdate types.Bool   `tfsdk:"redeploy_on_update"`
	Path             types.String `tfsdk:"path"`
	Status           types.String `tfsdk:"status"`
	ServiceCount     types.Int64  `tfsdk:"service_count"`
	RunningCount     types.Int64  `tfsdk:"running_count"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
	RemoveFiles      types.Bool   `tfsdk:"remove_files"`
	RemoveVolumes    types.Bool   `tfsdk:"remove_volumes"`
}

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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
			if err := r.client.UpProject(ctx, envID, out.ID); err != nil {
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

	state := projectModel{
		ID:            types.StringValue(out.ID),
		EnvironmentID: plan.EnvironmentID,
		Name:          types.StringValue(out.Name),
		Compose:       plan.Compose,
		Env:           plan.Env,
		Path:          types.StringValue(out.Path),
		Status:        types.StringValue(out.Status),
		ServiceCount:  types.Int64Value(int64(out.ServiceCount)),
		RunningCount:  types.Int64Value(int64(out.RunningCount)),
		CreatedAt:     types.StringValue(out.CreatedAt),
		UpdatedAt:     types.StringValue(out.UpdatedAt),
		RemoveFiles:   plan.RemoveFiles,
		RemoveVolumes: plan.RemoveVolumes,
	}
	state.Running = plan.Running
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
		if strings.Contains(strings.ToLower(err.Error()), "404") {
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
	state.CreatedAt = types.StringValue(out.CreatedAt)
	// Leave updated_at unchanged during Update to avoid plan inconsistency on server-side timestamp changes
	if out.ComposeContent != nil {
		state.Compose = types.StringValue(*out.ComposeContent)
	}
	if out.EnvContent != nil {
		state.Env = types.StringValue(*out.EnvContent)
	}

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

	envID := state.EnvironmentID.ValueString()
	projID := state.ID.ValueString()
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

	// Redeploy if compose/env changed and enabled (default true) and desired running true/unspecified
	changedContent := (body.ComposeContent != nil) || (body.EnvContent != nil)
	if changedContent {
		redeploy := true
		if !plan.RedeployOnUpdate.IsNull() && !plan.RedeployOnUpdate.IsUnknown() {
			redeploy = plan.RedeployOnUpdate.ValueBool()
		}
		runningDesired := true
		if !plan.Running.IsNull() && !plan.Running.IsUnknown() {
			runningDesired = plan.Running.ValueBool()
		}
		if redeploy && runningDesired {
			if err := r.client.RedeployProject(ctx, envID, projID); err != nil {
				resp.Diagnostics.AddError("project redeploy failed", err.Error())
				return
			}
			if det, derr := r.client.GetProject(ctx, envID, projID); derr == nil {
				out.Status = det.Status
			}
		}
	}

	// Lifecycle manage if configured
	if !plan.Running.IsNull() && !plan.Running.IsUnknown() {
		desired := plan.Running.ValueBool()
		current := state.Running.ValueBool()
		if desired != current {
			if desired {
				if err := r.client.UpProject(ctx, envID, projID); err != nil {
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
	state.CreatedAt = types.StringValue(out.CreatedAt)
	if out.ComposeContent != nil {
		state.Compose = types.StringValue(*out.ComposeContent)
	}
	if out.EnvContent != nil {
		state.Env = types.StringValue(*out.EnvContent)
	}
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
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("destroy project failed", err.Error())
	}
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
