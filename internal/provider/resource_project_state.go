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

var _ resource.Resource = &ProjectStateResource{}
var _ resource.ResourceWithImportState = &ProjectStateResource{}

type ProjectStateResource struct{ client *sdkclient.Client }

func NewProjectStateResource() resource.Resource { return &ProjectStateResource{} }

func (r *ProjectStateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_state"
}

func (r *ProjectStateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"id":             resourceschema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"environment_id": resourceschema.StringAttribute{Required: true, Description: "Environment ID"},
			"project_id":     resourceschema.StringAttribute{Required: true, Description: "Project ID"},
			"running":        resourceschema.BoolAttribute{Required: true, Description: "Whether the project should be running (docker compose up)"},
			"status":         resourceschema.StringAttribute{Computed: true, Description: "Project status string as returned by API"},
		},
	}
}

func (r *ProjectStateResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type projectStateModel struct {
	ID            types.String `tfsdk:"id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	ProjectID     types.String `tfsdk:"project_id"`
	Running       types.Bool   `tfsdk:"running"`
	Status        types.String `tfsdk:"status"`
}

func (r *ProjectStateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectStateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := plan.EnvironmentID.ValueString()
	projID := plan.ProjectID.ValueString()
	if plan.Running.ValueBool() {
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
	// read status
	out, err := r.client.GetProject(ctx, envID, projID)
	if err != nil {
		resp.Diagnostics.AddError("get project failed", err.Error())
		return
	}
	state := projectStateModel{
		ID:            types.StringValue(envID + ":" + projID),
		EnvironmentID: plan.EnvironmentID,
		ProjectID:     plan.ProjectID,
		Running:       plan.Running,
		Status:        types.StringValue(out.Status),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectStateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectStateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := state.EnvironmentID.ValueString()
	projID := state.ProjectID.ValueString()
	out, err := r.client.GetProject(ctx, envID, projID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("get project failed", err.Error())
		return
	}
	state.Status = types.StringValue(out.Status)
	// Running remains as configured; status is surfaced separately
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectStateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state projectStateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := state.EnvironmentID.ValueString()
	projID := state.ProjectID.ValueString()
	desired := plan.Running.ValueBool()
	if desired != state.Running.ValueBool() {
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
	}
	out, err := r.client.GetProject(ctx, envID, projID)
	if err != nil {
		resp.Diagnostics.AddError("get project failed", err.Error())
		return
	}
	state.Running = plan.Running
	state.Status = types.StringValue(out.Status)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectStateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No remote action; removing this resource does not stop the project
}

func (r *ProjectStateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// envID:projectID
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("invalid import id", "expected env_id:project_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
