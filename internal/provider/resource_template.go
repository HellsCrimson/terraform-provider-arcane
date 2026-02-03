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

var _ resource.Resource = &TemplateResource{}
var _ resource.ResourceWithImportState = &TemplateResource{}

type TemplateResource struct {
	client *sdkclient.Client
}

func NewTemplateResource() resource.Resource {
	return &TemplateResource{}
}

func (r *TemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template"
}

func (r *TemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manages a reusable docker-compose template.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the template",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": resourceschema.StringAttribute{
				Required:    true,
				Description: "Name of the template",
			},
			"description": resourceschema.StringAttribute{
				Required:    true,
				Description: "Description of the template",
			},
			"content": resourceschema.StringAttribute{
				Required:    true,
				Description: "Docker Compose YAML content",
			},
			"env_content": resourceschema.StringAttribute{
				Required:    true,
				Description: "Environment variables template content (.env format)",
			},
			"is_custom": resourceschema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is a custom template",
			},
			"is_remote": resourceschema.BoolAttribute{
				Computed:    true,
				Description: "Whether this template is from a remote registry",
			},
			"registry_id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "ID of the registry this template belongs to (if remote)",
			},
		},
	}
}

func (r *TemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type templateModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Content     types.String `tfsdk:"content"`
	EnvContent  types.String `tfsdk:"env_content"`
	IsCustom    types.Bool   `tfsdk:"is_custom"`
	IsRemote    types.Bool   `tfsdk:"is_remote"`
	RegistryID  types.String `tfsdk:"registry_id"`
}

func (r *TemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan templateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.CreateTemplateRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Content:     plan.Content.ValueString(),
		EnvContent:  plan.EnvContent.ValueString(),
	}

	template, err := r.client.CreateTemplate(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("create template failed", err.Error())
		return
	}

	state := templateModel{
		ID:          types.StringValue(template.ID),
		Name:        types.StringValue(template.Name),
		Description: types.StringValue(template.Description),
		Content:     types.StringValue(template.Content),
		IsCustom:    types.BoolValue(template.IsCustom),
		IsRemote:    types.BoolValue(template.IsRemote),
	}

	if template.EnvContent != nil {
		state.EnvContent = types.StringValue(*template.EnvContent)
	} else {
		state.EnvContent = types.StringNull()
	}
	if template.RegistryID != nil {
		state.RegistryID = types.StringValue(*template.RegistryID)
	} else {
		state.RegistryID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state templateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	template, err := r.client.GetTemplate(ctx, state.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read template failed", err.Error())
		return
	}

	state.Name = types.StringValue(template.Name)
	state.Description = types.StringValue(template.Description)
	state.Content = types.StringValue(template.Content)
	state.IsCustom = types.BoolValue(template.IsCustom)
	state.IsRemote = types.BoolValue(template.IsRemote)

	if template.EnvContent != nil {
		state.EnvContent = types.StringValue(*template.EnvContent)
	} else {
		state.EnvContent = types.StringNull()
	}
	if template.RegistryID != nil {
		state.RegistryID = types.StringValue(*template.RegistryID)
	} else {
		state.RegistryID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan templateModel
	var state templateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.UpdateTemplateRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Content:     plan.Content.ValueString(),
		EnvContent:  plan.EnvContent.ValueString(),
	}

	template, err := r.client.UpdateTemplate(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("update template failed", err.Error())
		return
	}

	state = plan
	state.ID = types.StringValue(template.ID)
	state.IsCustom = types.BoolValue(template.IsCustom)
	state.IsRemote = types.BoolValue(template.IsRemote)

	if template.EnvContent != nil {
		state.EnvContent = types.StringValue(*template.EnvContent)
	} else {
		state.EnvContent = types.StringNull()
	}
	if template.RegistryID != nil {
		state.RegistryID = types.StringValue(*template.RegistryID)
	} else {
		state.RegistryID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state templateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteTemplate(ctx, state.ID.ValueString()); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete template failed", err.Error())
	}
}

func (r *TemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
