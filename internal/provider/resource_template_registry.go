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

var _ resource.Resource = &TemplateRegistryResource{}
var _ resource.ResourceWithImportState = &TemplateRegistryResource{}

type TemplateRegistryResource struct {
	client *sdkclient.Client
}

func NewTemplateRegistryResource() resource.Resource {
	return &TemplateRegistryResource{}
}

func (r *TemplateRegistryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template_registry"
}

func (r *TemplateRegistryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manages an external template registry for accessing remote templates.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the template registry",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": resourceschema.StringAttribute{
				Required:    true,
				Description: "Name of the template registry",
			},
			"url": resourceschema.StringAttribute{
				Required:    true,
				Description: "URL of the template registry",
			},
			"description": resourceschema.StringAttribute{
				Required:    true,
				Description: "Description of the template registry",
			},
			"enabled": resourceschema.BoolAttribute{
				Required:    true,
				Description: "Whether the registry is enabled",
			},
		},
	}
}

func (r *TemplateRegistryResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type templateRegistryModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	URL         types.String `tfsdk:"url"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
}

func (r *TemplateRegistryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan templateRegistryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.CreateTemplateRegistryRequest{
		Name:        plan.Name.ValueString(),
		URL:         plan.URL.ValueString(),
		Description: plan.Description.ValueString(),
		Enabled:     plan.Enabled.ValueBool(),
	}

	registry, err := r.client.CreateTemplateRegistry(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("create template registry failed", err.Error())
		return
	}

	state := templateRegistryModel{
		ID:          types.StringValue(registry.ID),
		Name:        types.StringValue(registry.Name),
		URL:         types.StringValue(registry.URL),
		Description: types.StringValue(registry.Description),
		Enabled:     types.BoolValue(registry.Enabled),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TemplateRegistryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state templateRegistryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	registry, err := r.client.GetTemplateRegistry(ctx, state.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read template registry failed", err.Error())
		return
	}

	state.Name = types.StringValue(registry.Name)
	state.URL = types.StringValue(registry.URL)
	state.Description = types.StringValue(registry.Description)
	state.Enabled = types.BoolValue(registry.Enabled)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TemplateRegistryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan templateRegistryModel
	var state templateRegistryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.UpdateTemplateRegistryRequest{
		Name:        plan.Name.ValueString(),
		URL:         plan.URL.ValueString(),
		Description: plan.Description.ValueString(),
		Enabled:     plan.Enabled.ValueBool(),
	}

	registry, err := r.client.UpdateTemplateRegistry(ctx, state.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("update template registry failed", err.Error())
		return
	}

	state = plan
	state.ID = types.StringValue(registry.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TemplateRegistryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state templateRegistryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteTemplateRegistry(ctx, state.ID.ValueString()); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete template registry failed", err.Error())
	}
}

func (r *TemplateRegistryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
