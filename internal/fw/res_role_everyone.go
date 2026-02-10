package fw

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewRoleEveryoneResource() resource.Resource {
	return &roleEveryoneResource{}
}

type roleEveryoneResource struct {
	c *discord.RestClient
}

type roleEveryoneModel struct {
	ID types.String `tfsdk:"id"`

	ServerID types.String `tfsdk:"server_id"`

	Permissions       types.Int64  `tfsdk:"permissions"`
	PermissionsBits64 types.String `tfsdk:"permissions_bits64"`
}

func (r *roleEveryoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_everyone"
}

func (r *roleEveryoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"server_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permissions": schema.Int64Attribute{
				Optional: true,
			},
			"permissions_bits64": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Permissions as 64-bit integer string (decimal or 0x...). Prefer this for newer high-bit permissions.",
			},
		},
	}
}

func (r *roleEveryoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *roleEveryoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleEveryoneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Treat create as read (role always exists).
	state := plan
	r.readIntoState(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *roleEveryoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleEveryoneModel
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

func (r *roleEveryoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleEveryoneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := plan.ServerID.ValueString()
	plan.ID = types.StringValue(serverID)

	perms := uint64(0)
	if !plan.Permissions.IsNull() {
		perms = uint64(plan.Permissions.ValueInt64())
	}
	if s := strings.TrimSpace(plan.PermissionsBits64.ValueString()); s != "" {
		v, err := discord.Uint64StringToPermissionBit(s)
		if err != nil {
			resp.Diagnostics.AddError("Invalid permissions_bits64", err.Error())
			return
		}
		perms = v
	}

	body := restRoleUpdate{
		Permissions: strconv.FormatUint(perms, 10),
	}

	var out restRoleFull
	if err := r.c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s/roles/%s", serverID, serverID), nil, body, &out); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	state := plan
	r.readIntoState(ctx, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *roleEveryoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Deleting the everyone role is not allowed", "Destroying this resource removes it from state only.")
	resp.State.RemoveResource(ctx)
}

func (r *roleEveryoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format is just the server/guild ID (the @everyone role has the same ID as the guild).
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *roleEveryoneResource) readIntoState(ctx context.Context, state *roleEveryoneModel, diags discordFrameworkDiagnostics) {
	serverID := state.ServerID.ValueString()
	if serverID == "" {
		serverID = state.ID.ValueString()
	}
	state.ID = types.StringValue(serverID)
	state.ServerID = types.StringValue(serverID)

	role, err := fetchRoleByID(ctx, r.c, serverID, serverID)
	if err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	state.PermissionsBits64 = types.StringValue(strings.TrimSpace(role.Permissions))
	if v, err := discord.Uint64StringToPermissionBit(role.Permissions); err == nil {
		if i, err := discord.Uint64ToIntIfFits(v); err == nil {
			state.Permissions = types.Int64Value(int64(i))
		} else {
			state.Permissions = types.Int64Value(0)
		}
	}
}
