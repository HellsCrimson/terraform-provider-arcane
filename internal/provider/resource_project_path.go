package provider

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "os"
    "strings"

    "arcane-terraform/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ProjectPathResource{}
var _ resource.ResourceWithImportState = &ProjectPathResource{}
var _ resource.ResourceWithModifyPlan = &ProjectPathResource{}

type ProjectPathResource struct{ client *sdkclient.Client }

func NewProjectPathResource() resource.Resource { return &ProjectPathResource{} }

func (r *ProjectPathResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_path"
}

func (r *ProjectPathResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"id":             resourceschema.StringAttribute{Computed: true, Description: "Project ID", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"environment_id": resourceschema.StringAttribute{Required: true, Description: "Environment ID"},
			"name":           resourceschema.StringAttribute{Required: true, Description: "Project name"},
			"compose_path":   resourceschema.StringAttribute{Required: true, Description: "Filesystem path to docker-compose.yml"},
			"env_path":       resourceschema.StringAttribute{Optional: true, Description: "Filesystem path to .env"},

			// Controls whether to store full file contents or only hashes in state
			"content_hash_mode": resourceschema.BoolAttribute{Optional: true, Description: "If true, store only content hashes in state instead of full file contents."},

			// Derived from files; used to detect changes and apply updates
			"compose_content":      resourceschema.StringAttribute{Computed: true, Sensitive: true},
			"env_content":          resourceschema.StringAttribute{Computed: true, Sensitive: true},
			"compose_content_hash": resourceschema.StringAttribute{Computed: true, Sensitive: true},
			"env_content_hash":     resourceschema.StringAttribute{Computed: true, Sensitive: true},

			// Lifecycle (optional)
			"running": resourceschema.BoolAttribute{Optional: true, Description: "If true, ensure project is running (compose up); if false, compose down. If unset, no lifecycle management."},

			// Computed info
			"path":          resourceschema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"status":        resourceschema.StringAttribute{Computed: true},
			"service_count": resourceschema.Int64Attribute{Computed: true},
			"running_count": resourceschema.Int64Attribute{Computed: true},
			"created_at":    resourceschema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"updated_at":    resourceschema.StringAttribute{Computed: true},

			// Delete options
			"remove_files":   resourceschema.BoolAttribute{Optional: true, Description: "Remove files on destroy"},
			"remove_volumes": resourceschema.BoolAttribute{Optional: true, Description: "Remove volumes on destroy"},
		},
	}
}

func (r *ProjectPathResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type projectPathModel struct {
	ID              types.String `tfsdk:"id"`
	EnvironmentID   types.String `tfsdk:"environment_id"`
	Name            types.String `tfsdk:"name"`
	ComposePath     types.String `tfsdk:"compose_path"`
	EnvPath         types.String `tfsdk:"env_path"`
	ContentHashMode types.Bool   `tfsdk:"content_hash_mode"`
	Compose         types.String `tfsdk:"compose_content"`
	Env             types.String `tfsdk:"env_content"`
	ComposeHash     types.String `tfsdk:"compose_content_hash"`
	EnvHash         types.String `tfsdk:"env_content_hash"`
	Running         types.Bool   `tfsdk:"running"`
	Path            types.String `tfsdk:"path"`
	Status          types.String `tfsdk:"status"`
	ServiceCount    types.Int64  `tfsdk:"service_count"`
	RunningCount    types.Int64  `tfsdk:"running_count"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
	RemoveFiles     types.Bool   `tfsdk:"remove_files"`
	RemoveVolumes   types.Bool   `tfsdk:"remove_volumes"`
}

// ModifyPlan loads compose/env file contents into computed attributes so file changes are detected during planning.
func (r *ProjectPathResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || !req.Plan.Raw.IsKnown() {
		return
	}
	var plan projectPathModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.ComposePath.IsUnknown() || plan.ComposePath.IsNull() {
		return
	}

	composeBytes, err := os.ReadFile(plan.ComposePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("compose_path"), "read compose file failed", err.Error())
		return
	}
	// Compute and set content/hash depending on mode
	if plan.ContentHashMode.ValueBool() {
		h := sha256.Sum256(composeBytes)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("compose_content_hash"), hex.EncodeToString(h[:]))...)
	} else {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("compose_content"), string(composeBytes))...)
	}

	if !plan.EnvPath.IsNull() && !plan.EnvPath.IsUnknown() {
		b, err := os.ReadFile(plan.EnvPath.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("env_path"), "read env file failed", err.Error())
			return
		}
		if plan.ContentHashMode.ValueBool() {
			h := sha256.Sum256(b)
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("env_content_hash"), hex.EncodeToString(h[:]))...)
		} else {
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("env_content"), string(b))...)
		}
	}
}

func (r *ProjectPathResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectPathModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Always read from file for API body
	// Use contents from plan only if set (e.g., by ModifyPlan); otherwise read from path
	compose := plan.Compose.ValueString()
	if compose == "" {
		b, err := os.ReadFile(plan.ComposePath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("read compose file failed", err.Error())
			return
		}
		compose = string(b)
	}
	var envStr *string
	if !plan.EnvPath.IsNull() && !plan.EnvPath.IsUnknown() {
		b, err := os.ReadFile(plan.EnvPath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("read env file failed", err.Error())
			return
		}
		s := string(b)
		envStr = &s
	} else if !plan.Env.IsNull() && !plan.Env.IsUnknown() && plan.Env.ValueString() != "" {
		s := plan.Env.ValueString()
		envStr = &s
	}

    body := sdkclient.ProjectCreateRequest{Name: plan.Name.ValueString(), ComposeContent: compose, EnvContent: envStr}
    envID := plan.EnvironmentID.ValueString()
    out, err := r.client.CreateProject(ctx, envID, body)
    if err != nil {
        resp.Diagnostics.AddError("create project failed", err.Error())
        return
    }

    // Optionally manage lifecycle
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

    state := plan
    state.ID = types.StringValue(out.ID)
    state.Path = types.StringValue(out.Path)
    state.Status = types.StringValue(out.Status)
	state.ServiceCount = types.Int64Value(int64(out.ServiceCount))
	state.RunningCount = types.Int64Value(int64(out.RunningCount))
	state.CreatedAt = types.StringValue(out.CreatedAt)
	state.UpdatedAt = types.StringValue(out.UpdatedAt)
	if plan.ContentHashMode.ValueBool() {
		ch := sha256.Sum256([]byte(compose))
		state.ComposeHash = types.StringValue(hex.EncodeToString(ch[:]))
		if envStr != nil {
			eh := sha256.Sum256([]byte(*envStr))
			state.EnvHash = types.StringValue(hex.EncodeToString(eh[:]))
		} else {
			state.EnvHash = types.StringNull()
		}
		// Clear contents in state for hash mode
		state.Compose = types.StringNull()
		state.Env = types.StringNull()
	} else {
		state.Compose = types.StringValue(compose)
		if envStr != nil {
			state.Env = types.StringValue(*envStr)
		} else {
			state.Env = types.StringNull()
		}
	}
    state.Running = plan.Running
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectPathResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectPathModel
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
	state.UpdatedAt = types.StringValue(out.UpdatedAt)
	// retain Compose/Env from local files in state; do not overwrite from server
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectPathResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectPathModel
	var state projectPathModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envID := state.EnvironmentID.ValueString()
	projID := state.ID.ValueString()
	compose := plan.Compose.ValueString()
	if plan.ContentHashMode.ValueBool() || compose == "" {
		// In hash mode or when content missing, read from path
		b, err := os.ReadFile(plan.ComposePath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("read compose file failed", err.Error())
			return
		}
		compose = string(b)
	}
	var envStr *string
	if !plan.Env.IsNull() && !plan.Env.IsUnknown() && plan.Env.ValueString() != "" {
		s := plan.Env.ValueString()
		envStr = &s
	} else if !plan.EnvPath.IsNull() && !plan.EnvPath.IsUnknown() {
		b, err := os.ReadFile(plan.EnvPath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("read env file failed", err.Error())
			return
		}
		s := string(b)
		envStr = &s
	}

	body := sdkclient.ProjectUpdateRequest{}
	if compose != "" {
		body.ComposeContent = &compose
	}
	if envStr != nil {
		body.EnvContent = envStr
	}
	// name is required attr, may or may not change
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		n := plan.Name.ValueString()
		body.Name = &n
	}

	out, err := r.client.UpdateProject(ctx, envID, projID, body)
	if err != nil {
		resp.Diagnostics.AddError("update project failed", err.Error())
		return
	}

	state.Name = types.StringValue(out.Name)
	state.Path = types.StringValue(out.Path)
	state.Status = types.StringValue(out.Status)
	state.ServiceCount = types.Int64Value(int64(out.ServiceCount))
	state.RunningCount = types.Int64Value(int64(out.RunningCount))
	state.CreatedAt = types.StringValue(out.CreatedAt)
	state.UpdatedAt = types.StringValue(out.UpdatedAt)
	if plan.ContentHashMode.ValueBool() {
		ch := sha256.Sum256([]byte(compose))
		state.ComposeHash = types.StringValue(hex.EncodeToString(ch[:]))
		state.Compose = types.StringNull()
		if envStr != nil {
			eh := sha256.Sum256([]byte(*envStr))
			state.EnvHash = types.StringValue(hex.EncodeToString(eh[:]))
			state.Env = types.StringNull()
		} else {
			state.EnvHash = types.StringNull()
			state.Env = types.StringNull()
		}
	} else {
		if compose != "" {
			state.Compose = types.StringValue(compose)
		}
		if envStr != nil {
			state.Env = types.StringValue(*envStr)
		} else {
			state.Env = types.StringNull()
		}
	}
	// Lifecycle management if configured and changed
	if !plan.Running.IsNull() && !plan.Running.IsUnknown() {
		desired := plan.Running.ValueBool()
		current := state.Running.ValueBool()
		if desired != current {
			if desired {
				if err := r.client.UpProject(ctx, envID, projID); err != nil { resp.Diagnostics.AddError("project up failed", err.Error()); return }
			} else {
				if err := r.client.DownProject(ctx, envID, projID); err != nil { resp.Diagnostics.AddError("project down failed", err.Error()); return }
			}
			if det, derr := r.client.GetProject(ctx, envID, projID); derr == nil {
				state.Status = types.StringValue(det.Status)
			}
			state.Running = plan.Running
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectPathResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectPathModel
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

func (r *ProjectPathResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// envID:projectID
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("invalid import id", "expected env_id:project_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
