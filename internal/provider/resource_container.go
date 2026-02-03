package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ContainerResource{}
var _ resource.ResourceWithImportState = &ContainerResource{}

type ContainerResource struct{ client *sdkclient.Client }

func NewContainerResource() resource.Resource { return &ContainerResource{} }

func (r *ContainerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container"
}

func (r *ContainerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"id":             resourceschema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"environment_id": resourceschema.StringAttribute{Required: true, Description: "Environment ID"},
			"name":           resourceschema.StringAttribute{Required: true, Description: "Container name", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"image":          resourceschema.StringAttribute{Required: true, Description: "Image", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"auto_remove":    resourceschema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"command":        resourceschema.ListAttribute{Optional: true, ElementType: types.StringType, PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()}},
			"cpus":           resourceschema.Float64Attribute{Optional: true, PlanModifiers: []planmodifier.Float64{float64planmodifier.RequiresReplace()}},
			"entrypoint":     resourceschema.ListAttribute{Optional: true, ElementType: types.StringType, PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()}},
			"environment":    resourceschema.ListAttribute{Optional: true, ElementType: types.StringType, PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()}},
			"memory":         resourceschema.Int64Attribute{Optional: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
			"networks":       resourceschema.ListAttribute{Optional: true, ElementType: types.StringType, PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()}},
			"ports":          resourceschema.MapAttribute{Optional: true, ElementType: types.StringType, PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()}},
			"privileged":     resourceschema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"restart_policy": resourceschema.StringAttribute{Optional: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"user":           resourceschema.StringAttribute{Optional: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"volumes":        resourceschema.ListAttribute{Optional: true, ElementType: types.StringType, PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()}},
			"working_dir":    resourceschema.StringAttribute{Optional: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},

			// Additional container configuration
			"hostname":         resourceschema.StringAttribute{Optional: true, Description: "Container hostname", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"domain_name":      resourceschema.StringAttribute{Optional: true, Description: "Container domain name", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"labels":           resourceschema.MapAttribute{Optional: true, ElementType: types.StringType, Description: "Container labels for metadata", PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()}},
			"tty":              resourceschema.BoolAttribute{Optional: true, Description: "Allocate a pseudo-TTY", PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"attach_stdin":     resourceschema.BoolAttribute{Optional: true, Description: "Attach stdin", PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"attach_stdout":    resourceschema.BoolAttribute{Optional: true, Description: "Attach stdout", PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"attach_stderr":    resourceschema.BoolAttribute{Optional: true, Description: "Attach stderr", PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"open_stdin":       resourceschema.BoolAttribute{Optional: true, Description: "Keep stdin open", PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"stdin_once":       resourceschema.BoolAttribute{Optional: true, Description: "Close stdin after client disconnect", PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"network_disabled": resourceschema.BoolAttribute{Optional: true, Description: "Disable networking", PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},

			// Computed
			"created": resourceschema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"status":  resourceschema.StringAttribute{Computed: true},

			// Delete behavior
			"force_delete":   resourceschema.BoolAttribute{Optional: true, Description: "Force delete running container"},
			"remove_volumes": resourceschema.BoolAttribute{Optional: true, Description: "Remove volumes on delete"},
		},
	}
}

func (r *ContainerResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type containerModel struct {
	ID            types.String  `tfsdk:"id"`
	EnvironmentID types.String  `tfsdk:"environment_id"`
	Name          types.String  `tfsdk:"name"`
	Image         types.String  `tfsdk:"image"`
	AutoRemove    types.Bool    `tfsdk:"auto_remove"`
	Command       types.List    `tfsdk:"command"`
	CPUs          types.Float64 `tfsdk:"cpus"`
	Entrypoint    types.List    `tfsdk:"entrypoint"`
	Environment   types.List    `tfsdk:"environment"`
	Memory        types.Int64   `tfsdk:"memory"`
	Networks      types.List    `tfsdk:"networks"`
	Ports         types.Map     `tfsdk:"ports"`
	Privileged    types.Bool    `tfsdk:"privileged"`
	RestartPolicy types.String  `tfsdk:"restart_policy"`
	User          types.String  `tfsdk:"user"`
	Volumes       types.List    `tfsdk:"volumes"`
	WorkingDir    types.String  `tfsdk:"working_dir"`

	// Additional configuration
	Hostname        types.String `tfsdk:"hostname"`
	DomainName      types.String `tfsdk:"domain_name"`
	Labels          types.Map    `tfsdk:"labels"`
	TTY             types.Bool   `tfsdk:"tty"`
	AttachStdin     types.Bool   `tfsdk:"attach_stdin"`
	AttachStdout    types.Bool   `tfsdk:"attach_stdout"`
	AttachStderr    types.Bool   `tfsdk:"attach_stderr"`
	OpenStdin       types.Bool   `tfsdk:"open_stdin"`
	StdinOnce       types.Bool   `tfsdk:"stdin_once"`
	NetworkDisabled types.Bool   `tfsdk:"network_disabled"`

	Created types.String `tfsdk:"created"`
	Status  types.String `tfsdk:"status"`

	ForceDelete   types.Bool `tfsdk:"force_delete"`
	RemoveVolumes types.Bool `tfsdk:"remove_volumes"`
}

func (r *ContainerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan containerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	portMap := mapFromStringMap(ctx, plan.Ports)
	// Normalize keys/values like "8081/tcp" => "8081"
	if len(portMap) > 0 {
		portMap = normalizePortMap(portMap)
	}
	body := sdkclient.ContainerCreateRequest{
		Name:        plan.Name.ValueString(),
		Image:       plan.Image.ValueString(),
		Command:     listToStrings(ctx, plan.Command),
		Entrypoint:  listToStrings(ctx, plan.Entrypoint),
		Environment: listToStrings(ctx, plan.Environment),
		Networks:    listToStrings(ctx, plan.Networks),
		Volumes:     listToStrings(ctx, plan.Volumes),
		Ports:       portMap,
	}
	if !plan.AutoRemove.IsNull() && !plan.AutoRemove.IsUnknown() {
		v := plan.AutoRemove.ValueBool()
		body.AutoRemove = &v
	}
	if !plan.CPUs.IsNull() && !plan.CPUs.IsUnknown() {
		v := plan.CPUs.ValueFloat64()
		body.CPUs = &v
	}
	if !plan.Memory.IsNull() && !plan.Memory.IsUnknown() {
		v := plan.Memory.ValueInt64()
		body.Memory = &v
	}
	if !plan.Privileged.IsNull() && !plan.Privileged.IsUnknown() {
		v := plan.Privileged.ValueBool()
		body.Privileged = &v
	}
	if !plan.RestartPolicy.IsNull() && !plan.RestartPolicy.IsUnknown() {
		v := plan.RestartPolicy.ValueString()
		body.RestartPolicy = &v
	}
	if !plan.User.IsNull() && !plan.User.IsUnknown() {
		v := plan.User.ValueString()
		body.User = &v
	}
	if !plan.WorkingDir.IsNull() && !plan.WorkingDir.IsUnknown() {
		v := plan.WorkingDir.ValueString()
		body.WorkingDir = &v
	}
	if !plan.Hostname.IsNull() && !plan.Hostname.IsUnknown() {
		v := plan.Hostname.ValueString()
		body.Hostname = &v
	}
	if !plan.DomainName.IsNull() && !plan.DomainName.IsUnknown() {
		v := plan.DomainName.ValueString()
		body.Domainname = &v
	}
	if !plan.Labels.IsNull() && !plan.Labels.IsUnknown() {
		body.Labels = mapFromStringMap(ctx, plan.Labels)
	}
	if !plan.TTY.IsNull() && !plan.TTY.IsUnknown() {
		v := plan.TTY.ValueBool()
		body.TTY = &v
	}
	if !plan.AttachStdin.IsNull() && !plan.AttachStdin.IsUnknown() {
		v := plan.AttachStdin.ValueBool()
		body.AttachStdin = &v
	}
	if !plan.AttachStdout.IsNull() && !plan.AttachStdout.IsUnknown() {
		v := plan.AttachStdout.ValueBool()
		body.AttachStdout = &v
	}
	if !plan.AttachStderr.IsNull() && !plan.AttachStderr.IsUnknown() {
		v := plan.AttachStderr.ValueBool()
		body.AttachStderr = &v
	}
	if !plan.OpenStdin.IsNull() && !plan.OpenStdin.IsUnknown() {
		v := plan.OpenStdin.ValueBool()
		body.OpenStdin = &v
	}
	if !plan.StdinOnce.IsNull() && !plan.StdinOnce.IsUnknown() {
		v := plan.StdinOnce.ValueBool()
		body.StdinOnce = &v
	}
	if !plan.NetworkDisabled.IsNull() && !plan.NetworkDisabled.IsUnknown() {
		v := plan.NetworkDisabled.ValueBool()
		body.NetworkDisabled = &v
	}

	out, err := r.client.CreateContainer(ctx, plan.EnvironmentID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("create container failed", err.Error())
		return
	}

	state := plan
	state.ID = types.StringValue(out.ID)
	state.Created = types.StringValue(out.Created)
	state.Status = types.StringValue(out.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ContainerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state containerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := state.EnvironmentID.ValueString()
	id := state.ID.ValueString()
	out, err := r.client.GetContainer(ctx, envID, id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read container failed", err.Error())
		return
	}
	state.Name = types.StringValue(out.Name)
	state.Image = types.StringValue(out.Image)
	state.Created = types.StringValue(out.Created)
	state.Status = types.StringValue(out.Status)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ContainerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All changes force new via plan modifiers. Nothing to do.
	var state containerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ContainerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state containerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	envID := state.EnvironmentID.ValueString()
	id := state.ID.ValueString()
	force := state.ForceDelete.ValueBool()
	volumes := state.RemoveVolumes.ValueBool()
	if err := r.client.DeleteContainer(ctx, envID, id, force, volumes); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete container failed", err.Error())
	}
}

func (r *ContainerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// envID:containerID
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("invalid import id", "expected env_id:container_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// helpers
func listToStrings(ctx context.Context, v types.List) []string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	var out []string
	_ = v.ElementsAs(ctx, &out, false)
	return out
}

func mapFromStringMap(ctx context.Context, v types.Map) map[string]string {
	out := map[string]string{}
	if v.IsNull() || v.IsUnknown() {
		return out
	}
	var tmp map[string]string
	_ = v.ElementsAs(ctx, &tmp, false)
	for k, val := range tmp {
		out[k] = val
	}
	return out
}

func normalizePortMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	strip := func(s string) string {
		if i := strings.IndexByte(s, '/'); i >= 0 {
			return s[:i]
		}
		return s
	}
	for k, v := range in {
		out[strip(k)] = strip(v)
	}
	return out
}

func stringsToList(ctx context.Context, arr []string) types.List {
	if len(arr) == 0 {
		return types.ListNull(types.StringType)
	}
	list, _ := types.ListValueFrom(ctx, types.StringType, arr)
	return list
}
