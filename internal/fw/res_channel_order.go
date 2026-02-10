package fw

import (
	"context"
	"fmt"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw/planmod"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewChannelOrderResource() resource.Resource {
	return &channelOrderResource{}
}

type channelOrderResource struct {
	c *discord.RestClient
}

type channelOrderItemModel struct {
	ChannelID       types.String `tfsdk:"channel_id"`
	Position        types.Int64  `tfsdk:"position"`
	ParentID        types.String `tfsdk:"parent_id"`
	LockPermissions types.Bool   `tfsdk:"lock_permissions"`
}

type channelOrderModel struct {
	ID types.String `tfsdk:"id"`

	ServerID types.String            `tfsdk:"server_id"`
	Channel  []channelOrderItemModel `tfsdk:"channel"`
	Reason   types.String            `tfsdk:"reason"`
}

type restChannelPosition struct {
	ID              string `json:"id"`
	Position        int    `json:"position,omitempty"`
	ParentID        string `json:"parent_id,omitempty"`
	LockPermissions *bool  `json:"lock_permissions,omitempty"`
}

type restGuildChannel struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	ParentID string `json:"parent_id"`
}

func (r *channelOrderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel_order"
}

func (r *channelOrderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"server_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"channel": schema.ListNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"channel_id": schema.StringAttribute{Required: true},
						"position":   schema.Int64Attribute{Required: true},
						"parent_id":  schema.StringAttribute{Optional: true},
						"lock_permissions": schema.BoolAttribute{
							Optional: true,
						},
					},
				},
			},
			"reason": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					planmod.IgnoreChangesString(),
				},
				Description: "Optional audit log reason (X-Audit-Log-Reason). This value is not readable.",
			},
		},
	}
}

func (r *channelOrderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func expandChannelPositions(items []channelOrderItemModel) []restChannelPosition {
	out := make([]restChannelPosition, 0, len(items))
	for _, it := range items {
		p := restChannelPosition{
			ID:       it.ChannelID.ValueString(),
			Position: int(it.Position.ValueInt64()),
		}
		if !it.ParentID.IsNull() && it.ParentID.ValueString() != "" {
			p.ParentID = it.ParentID.ValueString()
		}
		if !it.LockPermissions.IsNull() {
			b := it.LockPermissions.ValueBool()
			p.LockPermissions = &b
		}
		out = append(out, p)
	}
	return out
}

func (r *channelOrderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan channelOrderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := expandChannelPositions(plan.Channel)
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/channels", plan.ServerID.ValueString()), nil, body, nil, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.ServerID.ValueString())
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *channelOrderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan channelOrderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := expandChannelPositions(plan.Channel)
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/channels", plan.ServerID.ValueString()), nil, body, nil, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.ServerID.ValueString())
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *channelOrderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state channelOrderModel
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

func (r *channelOrderResource) readIntoState(ctx context.Context, state *channelOrderModel, diags discordFrameworkDiagnostics) {
	serverID := state.ID.ValueString()
	if serverID == "" {
		serverID = state.ServerID.ValueString()
	}

	var channels []restGuildChannel
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/channels", serverID), nil, nil, &channels); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	index := map[string]restGuildChannel{}
	for _, ch := range channels {
		index[ch.ID] = ch
	}

	// Preserve config order, but refresh fields from remote for drift detection.
	out := make([]channelOrderItemModel, 0, len(state.Channel))
	for _, it := range state.Channel {
		id := it.ChannelID.ValueString()
		ch, ok := index[id]
		if !ok {
			diags.AddError("Channel not found", fmt.Sprintf("channel_id %s not found in server %s", id, serverID))
			return
		}
		out = append(out, channelOrderItemModel{
			ChannelID:       types.StringValue(id),
			Position:        types.Int64Value(int64(ch.Position)),
			ParentID:        types.StringValue(ch.ParentID),
			LockPermissions: it.LockPermissions,
		})
	}

	state.ID = types.StringValue(serverID)
	state.ServerID = types.StringValue(serverID)
	state.Channel = out
}

func (r *channelOrderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No-op. Ordering is not meaningfully "deletable".
	resp.Diagnostics.AddWarning("discord_channel_order does not revert ordering on destroy", "Destroying this resource removes it from state only.")
	resp.State.RemoveResource(ctx)
}

func (r *channelOrderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("server_id"), req, resp)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
