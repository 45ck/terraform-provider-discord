package fw

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw/fwutil"
	"github.com/aequasi/discord-terraform/internal/fw/planmod"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewBanResource() resource.Resource {
	return &banResource{}
}

type banResource struct {
	c *discord.RestClient
}

type banModel struct {
	ID types.String `tfsdk:"id"`

	ServerID types.String `tfsdk:"server_id"`
	UserID   types.String `tfsdk:"user_id"`

	DeleteMessageSeconds types.Int64  `tfsdk:"delete_message_seconds"`
	Reason               types.String `tfsdk:"reason"`
}

func (r *banResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ban"
}

func (r *banResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},

			"server_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"delete_message_seconds": schema.Int64Attribute{
				Optional:    true,
				Description: "How many seconds of messages to delete (0 to not delete). Create-only knob; Discord does not persist this setting for reads.",
				PlanModifiers: []planmodifier.Int64{
					// Create-only: treat changes as replace, matching SDK ForceNew.
					int64planmodifier.RequiresReplace(),
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

func (r *banResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func banID(serverID, userID string) string {
	return fmt.Sprintf("%s:%s", serverID, userID)
}

func (r *banResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan banModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := plan.ServerID.ValueString()
	userID := plan.UserID.ValueString()

	q := url.Values{}
	if !plan.DeleteMessageSeconds.IsNull() {
		q.Set("delete_message_seconds", fmt.Sprintf("%d", plan.DeleteMessageSeconds.ValueInt64()))
	}

	if err := r.c.DoJSONWithReason(ctx, "PUT", fmt.Sprintf("/guilds/%s/bans/%s", serverID, userID), q, nil, nil, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(banID(serverID, userID))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *banResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state banModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID, userID, err := fwutil.ParseTwoIDs(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", err.Error())
		return
	}

	var out any
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/bans/%s", serverID, userID), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	state.ServerID = types.StringValue(serverID)
	state.UserID = types.StringValue(userID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *banResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No Update: create-only; replace on change.
	resp.Diagnostics.AddError("Unsupported operation", "discord_ban does not support in-place updates")
}

func (r *banResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state banModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := state.ServerID.ValueString()
	userID := state.UserID.ValueString()
	if serverID == "" || userID == "" {
		var err error
		serverID, userID, err = fwutil.ParseTwoIDs(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid ID", err.Error())
			return
		}
	}

	if err := r.c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/guilds/%s/bans/%s", serverID, userID), nil, nil, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *banResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// ID is server_id:user_id
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
