package provider

import (
	"context"
	"strings"

	"terraform-provider-arcane/internal/sdkclient"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &UserDataSource{}

type UserDataSource struct {
	client *sdkclient.Client
}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

func (d *UserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source for reading an Arcane user",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "User ID",
			},
			"username": schema.StringAttribute{
				Computed:    true,
				Description: "Username",
			},
			"display_name": schema.StringAttribute{
				Computed:    true,
				Description: "Display name",
			},
			"email": schema.StringAttribute{
				Computed:    true,
				Description: "Email address",
			},
			"locale": schema.StringAttribute{
				Computed:    true,
				Description: "Locale preference",
			},
			"role_assignments": schema.SetNestedAttribute{
				Computed:    true,
				Description: "Role assignments held by the user.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role_id": schema.StringAttribute{
							Computed:    true,
							Description: "ID of the granted role.",
						},
						"environment_id": schema.StringAttribute{
							Computed:    true,
							Description: "Environment ID the assignment is scoped to; empty for a global assignment.",
						},
						"source": schema.StringAttribute{
							Computed:    true,
							Description: "How the assignment was created (manual or oidc).",
						},
					},
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
			},
		},
	}
}

func (d *UserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type userDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Username        types.String `tfsdk:"username"`
	DisplayName     types.String `tfsdk:"display_name"`
	Email           types.String `tfsdk:"email"`
	Locale          types.String `tfsdk:"locale"`
	RoleAssignments types.Set    `tfsdk:"role_assignments"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

type userDataSourceRoleAssignmentModel struct {
	RoleID        types.String `tfsdk:"role_id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	Source        types.String `tfsdk:"source"`
}

var userDataSourceRoleAssignmentType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"role_id":        types.StringType,
	"environment_id": types.StringType,
	"source":         types.StringType,
}}

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config userDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := d.client.GetUser(ctx, config.ID.ValueString())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "404") {
			resp.Diagnostics.AddError("user not found", "No user with id: "+config.ID.ValueString())
			return
		}
		resp.Diagnostics.AddError("failed to read user", err.Error())
		return
	}

	state := userDataSourceModel{
		ID:        types.StringValue(user.ID),
		Username:  types.StringValue(user.Username),
		CreatedAt: stringOrNull(user.CreatedAt),
		UpdatedAt: stringOrNull(user.UpdatedAt),
	}
	if user.Display != nil {
		state.DisplayName = types.StringValue(*user.Display)
	} else {
		state.DisplayName = types.StringNull()
	}
	if user.Email != nil {
		state.Email = types.StringValue(*user.Email)
	} else {
		state.Email = types.StringNull()
	}
	if user.Locale != nil {
		state.Locale = types.StringValue(*user.Locale)
	} else {
		state.Locale = types.StringNull()
	}
	assignments := make([]userDataSourceRoleAssignmentModel, 0, len(user.RoleAssignments))
	for _, a := range user.RoleAssignments {
		m := userDataSourceRoleAssignmentModel{
			RoleID: types.StringValue(a.RoleID),
			Source: types.StringValue(a.Source),
		}
		if a.EnvironmentID != "" {
			m.EnvironmentID = types.StringValue(a.EnvironmentID)
		} else {
			m.EnvironmentID = types.StringNull()
		}
		assignments = append(assignments, m)
	}
	roleSet, diags := types.SetValueFrom(ctx, userDataSourceRoleAssignmentType, assignments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.RoleAssignments = roleSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
