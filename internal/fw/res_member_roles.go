package fw

import (
	"context"
	"fmt"
	"strconv"

	"github.com/45ck/terraform-provider-discord/discord"
	"github.com/45ck/terraform-provider-discord/internal/fw/fwutil"
	"github.com/45ck/terraform-provider-discord/internal/fw/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewMemberRolesResource() resource.Resource {
	return &memberRolesResource{}
}

type memberRolesResource struct {
	c *discord.RestClient
}

type memberRoleItemModel struct {
	RoleID  types.String `tfsdk:"role_id"`
	HasRole types.Bool   `tfsdk:"has_role"`
}

type memberRolesModel struct {
	ID types.String `tfsdk:"id"`

	UserID   types.String          `tfsdk:"user_id"`
	ServerID types.String          `tfsdk:"server_id"`
	Role     []memberRoleItemModel `tfsdk:"role"`
}

type restMemberRoles struct {
	Roles []string `json:"roles"`
}

type restMemberForRoles struct {
	Roles []string `json:"roles"`
}

func (r *memberRolesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_member_roles"
}

func (r *memberRolesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"user_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"server_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"role": schema.SetNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role_id": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								validate.Snowflake(),
							},
						},
						"has_role": schema.BoolAttribute{
							Optional:    true,
							Description: "Whether the member should have this role. Defaults to true.",
						},
					},
				},
			},
		},
	}
}

func (r *memberRolesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func memberHasRole(roles []string, roleID string) bool {
	for _, rr := range roles {
		if rr == roleID {
			return true
		}
	}
	return false
}

func removeRoleID(roles []string, roleID string) []string {
	out := make([]string, 0, len(roles))
	for _, rr := range roles {
		if rr != roleID {
			out = append(out, rr)
		}
	}
	return out
}

func (r *memberRolesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan memberRolesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := plan.ServerID.ValueString()
	userID := plan.UserID.ValueString()

	// Validate member exists.
	var member restMemberForRoles
	if err := r.c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/members/"+userID, nil, nil, &member); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(discord.Hashcode(fmt.Sprintf("%s:%s", serverID, userID))))
	r.applyDesired(ctx, &plan, nil, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *memberRolesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state memberRolesModel
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

func (r *memberRolesResource) readIntoState(ctx context.Context, state *memberRolesModel, diags discordFrameworkDiagnostics) {
	serverID := state.ServerID.ValueString()
	userID := state.UserID.ValueString()

	var member restMemberForRoles
	if err := r.c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/members/"+userID, nil, nil, &member); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	// Refresh has_role from remote, preserving configured role_ids set.
	out := make([]memberRoleItemModel, 0, len(state.Role))
	for _, it := range state.Role {
		out = append(out, memberRoleItemModel{
			RoleID:  it.RoleID,
			HasRole: types.BoolValue(memberHasRole(member.Roles, it.RoleID.ValueString())),
		})
	}
	state.Role = out
}

func (r *memberRolesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan memberRolesModel
	var state memberRolesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.applyDesired(ctx, &plan, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func containsRole(items []memberRoleItemModel, roleID string) bool {
	for _, it := range items {
		if it.RoleID.ValueString() == roleID {
			return true
		}
	}
	return false
}

func desiredBoolDefaultTrue(v types.Bool) bool {
	if v.IsNull() || v.IsUnknown() {
		return true
	}
	return v.ValueBool()
}

func (r *memberRolesResource) applyDesired(ctx context.Context, plan *memberRolesModel, prior *memberRolesModel, diags discordFrameworkDiagnostics) {
	serverID := plan.ServerID.ValueString()
	userID := plan.UserID.ValueString()

	var member restMemberForRoles
	if err := r.c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/members/"+userID, nil, nil, &member); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			diags.AddError("Member not found", fmt.Sprintf("member %s not found in server %s", userID, serverID))
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	roles := member.Roles

	// Apply desired roles from plan.
	for _, it := range plan.Role {
		roleID := it.RoleID.ValueString()
		want := desiredBoolDefaultTrue(it.HasRole)
		has := memberHasRole(roles, roleID)
		if want && !has {
			roles = append(roles, roleID)
		}
		if !want && has {
			roles = removeRoleID(roles, roleID)
		}
	}

	// Remove roles that were removed from config (legacy behavior).
	if prior != nil {
		for _, it := range prior.Role {
			roleID := it.RoleID.ValueString()
			if containsRole(plan.Role, roleID) {
				continue
			}
			// If it was previously "has_role=true", then removing from config removes the role.
			if desiredBoolDefaultTrue(it.HasRole) {
				roles = removeRoleID(roles, roleID)
			}
		}
	}

	if err := r.c.DoJSON(ctx, "PATCH", "/guilds/"+serverID+"/members/"+userID, nil, restMemberRoles{Roles: roles}, nil); err != nil {
		diags.AddError("Discord API error", err.Error())
		return
	}
}

func (r *memberRolesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state memberRolesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := state.ServerID.ValueString()
	userID := state.UserID.ValueString()

	var member restMemberForRoles
	if err := r.c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/members/"+userID, nil, nil, &member); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	roles := member.Roles
	for _, it := range state.Role {
		roleID := it.RoleID.ValueString()
		if memberHasRole(roles, roleID) && desiredBoolDefaultTrue(it.HasRole) {
			roles = removeRoleID(roles, roleID)
		}
	}

	if err := r.c.DoJSON(ctx, "PATCH", "/guilds/"+serverID+"/members/"+userID, nil, restMemberRoles{Roles: roles}, nil); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *memberRolesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: server_id:user_id
	serverID, userID, err := fwutil.ParseTwoIDs(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected server_id:user_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), userID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(discord.Hashcode(fmt.Sprintf("%s:%s", serverID, userID))))...)
}
