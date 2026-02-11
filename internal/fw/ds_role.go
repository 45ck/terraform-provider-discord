package fw

import (
	"context"
	"strconv"
	"strings"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewRoleDataSource() datasource.DataSource {
	return &roleDataSource{}
}

type roleDataSource struct {
	c *discord.RestClient
}

type restRoleDS struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Hoist       bool   `json:"hoist"`
	Mentionable bool   `json:"mentionable"`
	Managed     bool   `json:"managed"`
	Position    int    `json:"position"`
	Permissions string `json:"permissions"`
}

type roleModel struct {
	ID                types.String `tfsdk:"id"`
	ServerID          types.String `tfsdk:"server_id"`
	RoleID            types.String `tfsdk:"role_id"`
	Name              types.String `tfsdk:"name"`
	Position          types.Int64  `tfsdk:"position"`
	Color             types.Int64  `tfsdk:"color"`
	Permissions       types.Int64  `tfsdk:"permissions"`
	PermissionsBits64 types.String `tfsdk:"permissions_bits64"`
	Hoist             types.Bool   `tfsdk:"hoist"`
	Mentionable       types.Bool   `tfsdk:"mentionable"`
	Managed           types.Bool   `tfsdk:"managed"`
}

func (d *roleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"server_id": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"role_id": schema.StringAttribute{
				Optional:    true,
				Description: "Either role_id or name must be set.",
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"name":     schema.StringAttribute{Optional: true, Description: "Either role_id or name must be set."},
			"position": schema.Int64Attribute{Computed: true},
			"color":    schema.Int64Attribute{Computed: true},
			"permissions": schema.Int64Attribute{
				Computed:    true,
				Description: "Role permission bits (platform-sized integer; can overflow on 32-bit). Prefer permissions_bits64.",
			},
			"permissions_bits64": schema.StringAttribute{
				Computed:    true,
				Description: "Permissions as 64-bit integer string (decimal). Prefer this for newer high-bit permissions.",
			},
			"hoist":       schema.BoolAttribute{Computed: true},
			"mentionable": schema.BoolAttribute{Computed: true},
			"managed":     schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *roleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.c = c.Rest
}

func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data roleModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := data.ServerID.ValueString()
	var roles []restRoleDS
	if err := d.c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/roles", nil, nil, &roles); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	var role *restRoleDS
	if !data.RoleID.IsNull() && data.RoleID.ValueString() != "" {
		id := data.RoleID.ValueString()
		for i := range roles {
			if roles[i].ID == id {
				role = &roles[i]
				break
			}
		}
		if role == nil {
			resp.Diagnostics.AddError("Role not found", "role_id was not found in the server")
			return
		}
	}

	if !data.Name.IsNull() && data.Name.ValueString() != "" {
		name := data.Name.ValueString()
		for i := range roles {
			if roles[i].Name == name {
				role = &roles[i]
				break
			}
		}
		if role == nil {
			resp.Diagnostics.AddError("Role not found", "name was not found in the server")
			return
		}
	}

	if role == nil {
		resp.Diagnostics.AddError("Invalid configuration", "either role_id or name must be set")
		return
	}

	permsRaw := strings.TrimSpace(role.Permissions)
	perms64, err := strconv.ParseUint(permsRaw, 10, 64)
	if err != nil {
		data.PermissionsBits64 = types.StringValue(permsRaw)
		data.Permissions = types.Int64Value(0)
	} else {
		data.PermissionsBits64 = types.StringValue(strconv.FormatUint(perms64, 10))
		if i, err := discord.Uint64ToIntIfFits(perms64); err == nil {
			data.Permissions = types.Int64Value(int64(i))
		} else {
			data.Permissions = types.Int64Value(0)
		}
	}

	data.ID = types.StringValue(role.ID)
	data.RoleID = types.StringValue(role.ID)
	data.Name = types.StringValue(role.Name)
	data.Position = types.Int64Value(int64(role.Position))
	data.Color = types.Int64Value(int64(role.Color))
	data.Hoist = types.BoolValue(role.Hoist)
	data.Mentionable = types.BoolValue(role.Mentionable)
	data.Managed = types.BoolValue(role.Managed)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
