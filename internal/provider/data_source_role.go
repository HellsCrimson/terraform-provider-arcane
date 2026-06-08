package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RoleDataSource{}

type RoleDataSource struct {
	client *sdkclient.Client
}

func NewRoleDataSource() datasource.DataSource {
	return &RoleDataSource{}
}

func (d *RoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *RoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an Arcane RBAC role by ID or name. Useful for resolving built-in role " +
			"IDs (e.g. for arcane_user.role_assignments) without hardcoding them.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Role ID. Provide either id or name.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Role name. Provide either id or name.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Optional human description.",
			},
			"permissions": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Permission strings granted by this role.",
			},
			"built_in": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is a built-in role.",
			},
			"assigned_user_count": schema.Int64Attribute{
				Computed:    true,
				Description: "How many users currently hold an assignment to this role.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp.",
			},
		},
	}
}

func (d *RoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type roleDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	Permissions       types.Set    `tfsdk:"permissions"`
	BuiltIn           types.Bool   `tfsdk:"built_in"`
	AssignedUserCount types.Int64  `tfsdk:"assigned_user_count"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (d *RoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config roleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != ""
	hasName := !config.Name.IsNull() && !config.Name.IsUnknown() && config.Name.ValueString() != ""
	if hasID == hasName {
		resp.Diagnostics.AddError("invalid role lookup", "Provide exactly one of `id` or `name`.")
		return
	}

	var role *sdkclient.Role
	if hasID {
		r, err := d.client.GetRole(ctx, config.ID.ValueString())
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "404") {
				resp.Diagnostics.AddError("role not found", "No role with id: "+config.ID.ValueString())
				return
			}
			resp.Diagnostics.AddError("failed to read role", err.Error())
			return
		}
		role = r
	} else {
		roles, err := d.client.ListRoles(ctx)
		if err != nil {
			resp.Diagnostics.AddError("failed to list roles", err.Error())
			return
		}
		name := config.Name.ValueString()
		for i := range roles {
			if roles[i].Name == name {
				role = &roles[i]
				break
			}
		}
		if role == nil {
			resp.Diagnostics.AddError("role not found", fmt.Sprintf("No role with name %q", name))
			return
		}
	}

	permSet, diags := roleStringSliceToSet(ctx, role.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state := roleDataSourceModel{
		ID:                types.StringValue(role.ID),
		Name:              types.StringValue(role.Name),
		Description:       stringOrNullEmpty(role.Description),
		Permissions:       permSet,
		BuiltIn:           types.BoolValue(role.BuiltIn),
		AssignedUserCount: types.Int64Value(role.AssignedUserCount),
		CreatedAt:         stringOrNullEmpty(role.CreatedAt),
		UpdatedAt:         stringOrNullEmpty(role.UpdatedAt),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
