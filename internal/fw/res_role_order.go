package fw

import (
	"context"
	"fmt"

	"github.com/45ck/terraform-provider-discord/discord"
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

func NewRoleOrderResource() resource.Resource {
	return &roleOrderResource{}
}

type roleOrderResource struct {
	c *discord.RestClient
}

type roleOrderItemModel struct {
	RoleID   types.String `tfsdk:"role_id"`
	Position types.Int64  `tfsdk:"position"`
}

type roleOrderModel struct {
	ID types.String `tfsdk:"id"`

	ServerID types.String         `tfsdk:"server_id"`
	Role     []roleOrderItemModel `tfsdk:"role"`
	Reason   types.String         `tfsdk:"reason"`
}

type restRoleOrderPosition struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

type restRoleOrder struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

func (r *roleOrderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_order"
}

func (r *roleOrderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"role": schema.ListNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role_id": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								validate.Snowflake(),
							},
						},
						"position": schema.Int64Attribute{Required: true},
					},
				},
			},
			"reason": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					planmod.IgnoreChangesString(),
				},
			},
		},
	}
}

func (r *roleOrderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func expandRolePositions(items []roleOrderItemModel) []restRoleOrderPosition {
	out := make([]restRoleOrderPosition, 0, len(items))
	for _, it := range items {
		out = append(out, restRoleOrderPosition{
			ID:       it.RoleID.ValueString(),
			Position: int(it.Position.ValueInt64()),
		})
	}
	return out
}

func (r *roleOrderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleOrderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := expandRolePositions(plan.Role)
	var out []restRoleOrder
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/roles", plan.ServerID.ValueString()), nil, body, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.ServerID.ValueString())
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleOrderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleOrderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := expandRolePositions(plan.Role)
	var out []restRoleOrder
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/roles", plan.ServerID.ValueString()), nil, body, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.ServerID.ValueString())
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleOrderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleOrderModel
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

func (r *roleOrderResource) readIntoState(ctx context.Context, state *roleOrderModel, diags discordFrameworkDiagnostics) {
	serverID := state.ID.ValueString()
	if serverID == "" {
		serverID = state.ServerID.ValueString()
	}

	var roles []restRoleOrder
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/roles", serverID), nil, nil, &roles); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	index := map[string]int{}
	for _, rr := range roles {
		index[rr.ID] = rr.Position
	}

	// Preserve config order, but refresh positions from remote for drift detection.
	out := make([]roleOrderItemModel, 0, len(state.Role))
	for _, it := range state.Role {
		id := it.RoleID.ValueString()
		pos, ok := index[id]
		if !ok {
			diags.AddError("Role not found", fmt.Sprintf("role_id %s not found in server %s", id, serverID))
			return
		}
		out = append(out, roleOrderItemModel{
			RoleID:   types.StringValue(id),
			Position: types.Int64Value(int64(pos)),
		})
	}

	state.ID = types.StringValue(serverID)
	state.ServerID = types.StringValue(serverID)
	state.Role = out
}

func (r *roleOrderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("discord_role_order does not revert ordering on destroy", "Destroying this resource removes it from state only.")
	resp.State.RemoveResource(ctx)
}

func (r *roleOrderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("server_id"), req, resp)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
