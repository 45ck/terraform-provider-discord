package fw

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/45ck/terraform-provider-discord/discord"
	"github.com/45ck/terraform-provider-discord/internal/fw/fwutil"
	"github.com/45ck/terraform-provider-discord/internal/fw/planmod"
	"github.com/45ck/terraform-provider-discord/internal/fw/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewRoleResource() resource.Resource {
	return &roleResource{}
}

type roleResource struct {
	c *discord.RestClient
}

type roleResourceModel struct {
	ID types.String `tfsdk:"id"`

	ServerID types.String `tfsdk:"server_id"`
	Name     types.String `tfsdk:"name"`
	Reason   types.String `tfsdk:"reason"`

	Permissions       types.Int64  `tfsdk:"permissions"`
	PermissionsBits64 types.String `tfsdk:"permissions_bits64"`

	Color       types.Int64 `tfsdk:"color"`
	Hoist       types.Bool  `tfsdk:"hoist"`
	Mentionable types.Bool  `tfsdk:"mentionable"`
	Position    types.Int64 `tfsdk:"position"`
	Managed     types.Bool  `tfsdk:"managed"`
}

type restRoleFull struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Hoist       bool   `json:"hoist"`
	Mentionable bool   `json:"mentionable"`
	Managed     bool   `json:"managed"`
	Position    int    `json:"position"`
	Permissions string `json:"permissions"`
}

type restRoleUpdate struct {
	Name        string `json:"name,omitempty"`
	Permissions string `json:"permissions,omitempty"`
	Color       int    `json:"color,omitempty"`
	Hoist       bool   `json:"hoist,omitempty"`
	Mentionable bool   `json:"mentionable,omitempty"`
}

type restRolePosition struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

type restRole struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},

			"server_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"name": schema.StringAttribute{Required: true},
			"reason": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					planmod.IgnoreChangesString(),
				},
				Description: "Optional audit log reason (X-Audit-Log-Reason). This value is not readable.",
			},

			"permissions": schema.Int64Attribute{
				Optional: true,
				// Historically TypeInt with default 0; Int64 is safer.
			},
			"permissions_bits64": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Permissions as 64-bit integer string (decimal or 0x...). Prefer this for newer high-bit permissions.",
			},

			"color":       schema.Int64Attribute{Optional: true},
			"hoist":       schema.BoolAttribute{Optional: true},
			"mentionable": schema.BoolAttribute{Optional: true},
			"position":    schema.Int64Attribute{Optional: true},

			"managed": schema.BoolAttribute{Computed: true},
		},
	}
}

func (r *roleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func desiredPerms64FromModel(m roleResourceModel) (uint64, error) {
	perms := uint64(0)
	if !m.Permissions.IsNull() {
		perms = uint64(m.Permissions.ValueInt64())
	}
	if s := strings.TrimSpace(m.PermissionsBits64.ValueString()); s != "" {
		v, err := discord.Uint64StringToPermissionBit(s)
		if err != nil {
			return 0, fmt.Errorf("invalid permissions_bits64: %w", err)
		}
		perms = v
	}
	return perms, nil
}

func fetchRoleByID(ctx context.Context, c *discord.RestClient, serverID, roleID string) (*restRoleFull, error) {
	var roles []restRoleFull
	if err := c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/roles", nil, nil, &roles); err != nil {
		return nil, err
	}
	for i := range roles {
		if roles[i].ID == roleID {
			return &roles[i], nil
		}
	}
	return nil, &discord.DiscordHTTPError{Method: "GET", Path: "/guilds/" + serverID + "/roles", StatusCode: 404, Message: "role not found"}
}

func swapRolePosition(ctx context.Context, c *discord.RestClient, serverID, roleID string, newPos int, reason string) error {
	var roles []restRoleFull
	if err := c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/roles", nil, nil, &roles); err != nil {
		return err
	}

	var current *restRoleFull
	var occupant *restRoleFull
	for i := range roles {
		if roles[i].ID == roleID {
			current = &roles[i]
		}
		if roles[i].Position == newPos {
			occupant = &roles[i]
		}
	}
	if current == nil {
		return fmt.Errorf("role %s not found in server %s", roleID, serverID)
	}
	if occupant == nil {
		return fmt.Errorf("new role position is out of bounds: %d", newPos)
	}
	if occupant.ID == roleID {
		return nil
	}

	body := []restRolePosition{
		{ID: occupant.ID, Position: current.Position},
		{ID: roleID, Position: newPos},
	}
	var out []restRole
	return c.DoJSONWithReason(ctx, "PATCH", "/guilds/"+serverID+"/roles", nil, body, &out, reason)
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := plan.ServerID.ValueString()

	perms, err := desiredPerms64FromModel(plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid permissions", err.Error())
		return
	}

	create := restRoleUpdate{
		Name:        plan.Name.ValueString(),
		Permissions: strconv.FormatUint(perms, 10),
		Color:       int(plan.Color.ValueInt64()),
		Hoist:       !plan.Hoist.IsNull() && plan.Hoist.ValueBool(),
		Mentionable: !plan.Mentionable.IsNull() && plan.Mentionable.ValueBool(),
	}

	var role restRoleFull
	if err := r.c.DoJSONWithReason(ctx, "POST", "/guilds/"+serverID+"/roles", nil, create, &role, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(role.ID)
	plan.Managed = types.BoolValue(role.Managed)
	plan.PermissionsBits64 = types.StringValue(strings.TrimSpace(role.Permissions))

	if !plan.Position.IsNull() {
		if err := swapRolePosition(ctx, r.c, serverID, role.ID, int(plan.Position.ValueInt64()), plan.Reason.ValueString()); err != nil {
			resp.Diagnostics.AddError("Discord API error", err.Error())
			return
		}
	}

	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readIntoState(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleResourceModel
	var state roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := state.ServerID.ValueString()
	roleID := state.ID.ValueString()

	if fwutil.ChangedInt64(plan.Position, state.Position) && !plan.Position.IsNull() {
		if err := swapRolePosition(ctx, r.c, serverID, roleID, int(plan.Position.ValueInt64()), plan.Reason.ValueString()); err != nil {
			resp.Diagnostics.AddError("Discord API error", err.Error())
			return
		}
	}

	perms, err := desiredPerms64FromModel(plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid permissions", err.Error())
		return
	}

	update := restRoleUpdate{
		Name:        plan.Name.ValueString(),
		Permissions: strconv.FormatUint(perms, 10),
		Color:       int(plan.Color.ValueInt64()),
		Hoist:       !plan.Hoist.IsNull() && plan.Hoist.ValueBool(),
		Mentionable: !plan.Mentionable.IsNull() && plan.Mentionable.ValueBool(),
	}

	var out restRoleFull
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/roles/%s", serverID, roleID), nil, update, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = state.ID
	plan.ServerID = state.ServerID
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := state.ServerID.ValueString()
	roleID := state.ID.ValueString()

	if err := r.c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/guilds/%s/roles/%s", serverID, roleID), nil, nil, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: server_id:role_id
	serverID, roleID, err := fwutil.ParseTwoIDs(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), roleID)...)
}

func (r *roleResource) readIntoState(ctx context.Context, state *roleResourceModel, diags discordFrameworkDiagnostics) {
	serverID := state.ServerID.ValueString()
	roleID := state.ID.ValueString()

	role, err := fetchRoleByID(ctx, r.c, serverID, roleID)
	if err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	state.Name = types.StringValue(role.Name)
	state.Position = types.Int64Value(int64(role.Position))
	state.Color = types.Int64Value(int64(role.Color))
	state.Hoist = types.BoolValue(role.Hoist)
	state.Mentionable = types.BoolValue(role.Mentionable)
	state.Managed = types.BoolValue(role.Managed)
	state.PermissionsBits64 = types.StringValue(strings.TrimSpace(role.Permissions))

	if v, err := discord.Uint64StringToPermissionBit(role.Permissions); err == nil {
		if i, err := discord.Uint64ToIntIfFits(v); err == nil {
			state.Permissions = types.Int64Value(int64(i))
		} else {
			state.Permissions = types.Int64Value(0)
		}
	}
}
