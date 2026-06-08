package provider

import (
	"context"
	"sort"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RolePermissionsDataSource{}

type RolePermissionsDataSource struct {
	client *sdkclient.Client
}

func NewRolePermissionsDataSource() datasource.DataSource {
	return &RolePermissionsDataSource{}
}

func (d *RolePermissionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_permissions"
}

func (d *RolePermissionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns the permission manifest: every permission the server recognizes, grouped " +
			"by resource, plus preset bundles. Useful for building arcane_role / arcane_api_key permission lists.",
		Attributes: map[string]schema.Attribute{
			"all_permissions": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Flattened, sorted set of every permission string the server recognizes.",
			},
			"resources": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Permission groups, in display order.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key":   schema.StringAttribute{Computed: true, Description: "Stable resource key (e.g. containers)."},
						"label": schema.StringAttribute{Computed: true, Description: "Human-readable label."},
						"scope": schema.StringAttribute{Computed: true, Description: "'global' or 'env'."},
						"permissions": schema.SetAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "Permission strings belonging to this resource.",
						},
					},
				},
			},
			"presets": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Optional preset permission bundles for bulk selection.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key":   schema.StringAttribute{Computed: true, Description: "Stable preset key."},
						"label": schema.StringAttribute{Computed: true, Description: "Human-readable preset label."},
						"permissions": schema.SetAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "Permissions included in the preset.",
						},
					},
				},
			},
		},
	}
}

func (d *RolePermissionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*sdkclient.Client)
	if !ok {
		resp.Diagnostics.AddError("unexpected provider data type", "Expected *sdkclient.Client")
		return
	}
	d.client = client
}

type rolePermissionsResourceModel struct {
	Key         types.String `tfsdk:"key"`
	Label       types.String `tfsdk:"label"`
	Scope       types.String `tfsdk:"scope"`
	Permissions types.Set    `tfsdk:"permissions"`
}

type rolePermissionsPresetModel struct {
	Key         types.String `tfsdk:"key"`
	Label       types.String `tfsdk:"label"`
	Permissions types.Set    `tfsdk:"permissions"`
}

type rolePermissionsDataSourceModel struct {
	AllPermissions types.Set  `tfsdk:"all_permissions"`
	Resources      types.List `tfsdk:"resources"`
	Presets        types.List `tfsdk:"presets"`
}

var rolePermissionsResourceObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"key":         types.StringType,
	"label":       types.StringType,
	"scope":       types.StringType,
	"permissions": types.SetType{ElemType: types.StringType},
}}

var rolePermissionsPresetObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"key":         types.StringType,
	"label":       types.StringType,
	"permissions": types.SetType{ElemType: types.StringType},
}}

func (d *RolePermissionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	manifest, err := d.client.GetPermissionsManifest(ctx)
	if err != nil {
		resp.Diagnostics.AddError("failed to read permission manifest", err.Error())
		return
	}

	allSet := map[string]struct{}{}
	resources := make([]rolePermissionsResourceModel, 0, len(manifest.Resources))
	for _, r := range manifest.Resources {
		perms := make([]string, 0, len(r.Actions))
		for _, a := range r.Actions {
			perms = append(perms, a.Permission)
			allSet[a.Permission] = struct{}{}
		}
		sort.Strings(perms)
		permSet, diags := roleStringSliceToSet(ctx, perms)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		resources = append(resources, rolePermissionsResourceModel{
			Key:         types.StringValue(r.Key),
			Label:       types.StringValue(r.Label),
			Scope:       types.StringValue(r.Scope),
			Permissions: permSet,
		})
	}

	presets := make([]rolePermissionsPresetModel, 0, len(manifest.Presets))
	for _, p := range manifest.Presets {
		permSet, diags := roleStringSliceToSet(ctx, p.Permissions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		presets = append(presets, rolePermissionsPresetModel{
			Key:         types.StringValue(p.Key),
			Label:       types.StringValue(p.Label),
			Permissions: permSet,
		})
	}

	all := make([]string, 0, len(allSet))
	for p := range allSet {
		all = append(all, p)
	}
	sort.Strings(all)

	allPermSet, diags := roleStringSliceToSet(ctx, all)
	resp.Diagnostics.Append(diags...)
	resourcesList, d1 := types.ListValueFrom(ctx, rolePermissionsResourceObjectType, resources)
	resp.Diagnostics.Append(d1...)
	presetsList, d2 := types.ListValueFrom(ctx, rolePermissionsPresetObjectType, presets)
	resp.Diagnostics.Append(d2...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := rolePermissionsDataSourceModel{
		AllPermissions: allPermSet,
		Resources:      resourcesList,
		Presets:        presetsList,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
