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

var _ resource.Resource = &RegistryResource{}
var _ resource.ResourceWithImportState = &RegistryResource{}

type RegistryResource struct{ client *sdkclient.Client }

func NewRegistryResource() resource.Resource { return &RegistryResource{} }

func (r *RegistryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry"
}

func (r *RegistryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"id":                    resourceschema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"url":                   resourceschema.StringAttribute{Required: true, Description: "Registry URL"},
			"username":              resourceschema.StringAttribute{Required: true, Description: "Registry username"},
			"token":                 resourceschema.StringAttribute{Required: true, Sensitive: true, Description: "Registry access token or password"},
			"description":           resourceschema.StringAttribute{Optional: true},
			"insecure":              resourceschema.BoolAttribute{Optional: true},
			"enabled":               resourceschema.BoolAttribute{Optional: true},
			"registry_type":         resourceschema.StringAttribute{Optional: true, Description: "Registry implementation type"},
			"aws_access_key_id":     resourceschema.StringAttribute{Optional: true, Sensitive: true},
			"aws_secret_access_key": resourceschema.StringAttribute{Optional: true, Sensitive: true},
			"aws_region":            resourceschema.StringAttribute{Optional: true},

			// Computed timestamps
			"created_at": resourceschema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"updated_at": resourceschema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		},
	}
}

func (r *RegistryResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		if c, ok := req.ProviderData.(*sdkclient.Client); ok {
			r.client = c
		}
	}
}

type registryModel struct {
	ID                 types.String `tfsdk:"id"`
	URL                types.String `tfsdk:"url"`
	Username           types.String `tfsdk:"username"`
	Token              types.String `tfsdk:"token"`
	Description        types.String `tfsdk:"description"`
	Insecure           types.Bool   `tfsdk:"insecure"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	RegistryType       types.String `tfsdk:"registry_type"`
	AWSAccessKeyID     types.String `tfsdk:"aws_access_key_id"`
	AWSSecretAccessKey types.String `tfsdk:"aws_secret_access_key"`
	AWSRegion          types.String `tfsdk:"aws_region"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (r *RegistryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan registryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdkclient.CreateContainerRegistryRequest{
		URL:      plan.URL.ValueString(),
		Username: plan.Username.ValueString(),
		Token:    plan.Token.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		body.Description = &v
	}
	if !plan.Insecure.IsNull() && !plan.Insecure.IsUnknown() {
		v := plan.Insecure.ValueBool()
		body.Insecure = &v
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		body.Enabled = &v
	}
	if !plan.RegistryType.IsNull() && !plan.RegistryType.IsUnknown() {
		v := plan.RegistryType.ValueString()
		body.RegistryType = &v
	}
	if !plan.AWSAccessKeyID.IsNull() && !plan.AWSAccessKeyID.IsUnknown() {
		v := plan.AWSAccessKeyID.ValueString()
		body.AWSAccessKeyID = &v
	}
	if !plan.AWSSecretAccessKey.IsNull() && !plan.AWSSecretAccessKey.IsUnknown() {
		v := plan.AWSSecretAccessKey.ValueString()
		body.AWSSecretAccessKey = &v
	}
	if !plan.AWSRegion.IsNull() && !plan.AWSRegion.IsUnknown() {
		v := plan.AWSRegion.ValueString()
		body.AWSRegion = &v
	}

	reg, err := r.client.CreateContainerRegistry(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("create registry failed", err.Error())
		return
	}

	state := registryModel{
		ID:                 types.StringValue(reg.ID),
		URL:                types.StringValue(reg.URL),
		Username:           plan.Username,
		Token:              plan.Token, // keep token in state for apply consistency
		Description:        plan.Description,
		Insecure:           plan.Insecure,
		Enabled:            plan.Enabled,
		RegistryType:       plan.RegistryType,
		AWSAccessKeyID:     plan.AWSAccessKeyID,
		AWSSecretAccessKey: plan.AWSSecretAccessKey,
		AWSRegion:          plan.AWSRegion,
		CreatedAt:          types.StringValue(reg.CreatedAt),
		UpdatedAt:          types.StringValue(reg.UpdatedAt),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RegistryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state registryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	reg, err := r.client.GetContainerRegistry(ctx, id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("read registry failed", err.Error())
		return
	}

	state.URL = types.StringValue(reg.URL)
	state.Username = types.StringValue(reg.Username)
	state.Description = types.StringValue(reg.Description)
	state.Insecure = types.BoolValue(reg.Insecure)
	state.Enabled = types.BoolValue(reg.Enabled)
	state.RegistryType = types.StringValue(reg.RegistryType)
	state.AWSAccessKeyID = state.AWSAccessKeyID
	state.AWSSecretAccessKey = state.AWSSecretAccessKey
	state.AWSRegion = types.StringValue(reg.AWSRegion)
	state.CreatedAt = types.StringValue(reg.CreatedAt)
	state.UpdatedAt = types.StringValue(reg.UpdatedAt)
	// Token remains as last set
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RegistryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state registryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	body := sdkclient.UpdateContainerRegistryRequest{}
	if !plan.URL.IsNull() && !plan.URL.IsUnknown() {
		v := plan.URL.ValueString()
		body.URL = &v
	}
	if !plan.Username.IsNull() && !plan.Username.IsUnknown() {
		v := plan.Username.ValueString()
		body.Username = &v
	}
	if !plan.Token.IsNull() && !plan.Token.IsUnknown() && plan.Token.ValueString() != "" {
		v := plan.Token.ValueString()
		body.Token = &v
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		body.Description = &v
	}
	if !plan.Insecure.IsNull() && !plan.Insecure.IsUnknown() {
		v := plan.Insecure.ValueBool()
		body.Insecure = &v
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		body.Enabled = &v
	}
	if !plan.RegistryType.IsNull() && !plan.RegistryType.IsUnknown() {
		v := plan.RegistryType.ValueString()
		body.RegistryType = &v
	}
	if !plan.AWSAccessKeyID.IsNull() && !plan.AWSAccessKeyID.IsUnknown() {
		v := plan.AWSAccessKeyID.ValueString()
		body.AWSAccessKeyID = &v
	}
	if !plan.AWSSecretAccessKey.IsNull() && !plan.AWSSecretAccessKey.IsUnknown() {
		v := plan.AWSSecretAccessKey.ValueString()
		body.AWSSecretAccessKey = &v
	}
	if !plan.AWSRegion.IsNull() && !plan.AWSRegion.IsUnknown() {
		v := plan.AWSRegion.ValueString()
		body.AWSRegion = &v
	}

	reg, err := r.client.UpdateContainerRegistry(ctx, id, body)
	if err != nil {
		resp.Diagnostics.AddError("update registry failed", err.Error())
		return
	}

	state.URL = types.StringValue(reg.URL)
	state.Username = types.StringValue(reg.Username)
	state.Description = types.StringValue(reg.Description)
	state.Insecure = types.BoolValue(reg.Insecure)
	state.Enabled = types.BoolValue(reg.Enabled)
	state.RegistryType = types.StringValue(reg.RegistryType)
	if !plan.AWSAccessKeyID.IsNull() && !plan.AWSAccessKeyID.IsUnknown() {
		state.AWSAccessKeyID = plan.AWSAccessKeyID
	}
	if !plan.AWSSecretAccessKey.IsNull() && !plan.AWSSecretAccessKey.IsUnknown() {
		state.AWSSecretAccessKey = plan.AWSSecretAccessKey
	}
	state.AWSRegion = types.StringValue(reg.AWSRegion)
	state.CreatedAt = types.StringValue(reg.CreatedAt)
	state.UpdatedAt = types.StringValue(reg.UpdatedAt)
	// Keep token in state if provided
	if !plan.Token.IsNull() && !plan.Token.IsUnknown() && plan.Token.ValueString() != "" {
		state.Token = plan.Token
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RegistryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state registryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	if err := r.client.DeleteContainerRegistry(ctx, id); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			return
		}
		resp.Diagnostics.AddError("delete registry failed", err.Error())
	}
}

func (r *RegistryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by ID
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
