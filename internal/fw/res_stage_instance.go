package fw

import (
	"context"
	"fmt"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewStageInstanceResource() resource.Resource {
	return &stageInstanceResource{}
}

type stageInstanceResource struct {
	c *discord.RestClient
}

type restStageInstance struct {
	ID                    string `json:"id"`
	ChannelID             string `json:"channel_id"`
	GuildID               string `json:"guild_id"`
	Topic                 string `json:"topic"`
	PrivacyLevel          int    `json:"privacy_level"`
	GuildScheduledEventID string `json:"guild_scheduled_event_id"`
}

type stageInstanceModel struct {
	ID types.String `tfsdk:"id"`

	ChannelID types.String `tfsdk:"channel_id"`
	Topic     types.String `tfsdk:"topic"`

	PrivacyLevel types.Int64 `tfsdk:"privacy_level"`

	SendStartNotification types.Bool   `tfsdk:"send_start_notification"`
	ScheduledEventID      types.String `tfsdk:"scheduled_event_id"`

	ServerID types.String `tfsdk:"server_id"`
}

func (r *stageInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stage_instance"
}

func (r *stageInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"channel_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Stage channel ID. Stage instances are keyed by channel_id in the Discord API.",
			},
			"topic": schema.StringAttribute{
				Required: true,
			},
			"privacy_level": schema.Int64Attribute{
				Optional: true,
				Default:  int64default.StaticInt64(2),
			},
			"send_start_notification": schema.BoolAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"scheduled_event_id": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"server_id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *stageInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *stageInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan stageInstanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"channel_id":    plan.ChannelID.ValueString(),
		"topic":         plan.Topic.ValueString(),
		"privacy_level": int(plan.PrivacyLevel.ValueInt64()),
	}
	if !plan.SendStartNotification.IsNull() && !plan.SendStartNotification.IsUnknown() {
		body["send_start_notification"] = plan.SendStartNotification.ValueBool()
	}
	if v := plan.ScheduledEventID.ValueString(); v != "" {
		body["guild_scheduled_event_id"] = v
	}

	var out restStageInstance
	if err := r.c.DoJSON(ctx, "POST", "/stage-instances", nil, body, &out); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.ChannelID.ValueString())
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *stageInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state stageInstanceModel
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

func (r *stageInstanceResource) readIntoState(ctx context.Context, state *stageInstanceModel, diags discordFrameworkDiagnostics) {
	channelID := state.ID.ValueString()
	if channelID == "" {
		channelID = state.ChannelID.ValueString()
	}
	if channelID == "" {
		state.ID = types.StringNull()
		return
	}

	var out restStageInstance
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/stage-instances/%s", channelID), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	state.ID = types.StringValue(out.ChannelID)
	state.ChannelID = types.StringValue(out.ChannelID)
	state.Topic = types.StringValue(out.Topic)
	state.PrivacyLevel = types.Int64Value(int64(out.PrivacyLevel))
	state.ScheduledEventID = types.StringValue(out.GuildScheduledEventID)
	state.ServerID = types.StringValue(out.GuildID)
}

func (r *stageInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan stageInstanceModel
	var prior stageInstanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}
	if plan.Topic.ValueString() != prior.Topic.ValueString() {
		body["topic"] = plan.Topic.ValueString()
	}
	if !plan.PrivacyLevel.IsNull() && !prior.PrivacyLevel.IsNull() && plan.PrivacyLevel.ValueInt64() != prior.PrivacyLevel.ValueInt64() {
		body["privacy_level"] = int(plan.PrivacyLevel.ValueInt64())
	}

	if len(body) > 0 {
		if err := r.c.DoJSON(ctx, "PATCH", fmt.Sprintf("/stage-instances/%s", prior.ChannelID.ValueString()), nil, body, nil); err != nil {
			resp.Diagnostics.AddError("Discord API error", err.Error())
			return
		}
	}

	plan.ID = types.StringValue(prior.ChannelID.ValueString())
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *stageInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state stageInstanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channelID := state.ChannelID.ValueString()
	if channelID == "" {
		channelID = state.ID.ValueString()
	}

	if err := r.c.DoJSON(ctx, "DELETE", fmt.Sprintf("/stage-instances/%s", channelID), nil, nil, nil); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *stageInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID is the channel_id.
	resource.ImportStatePassthroughID(ctx, path.Root("channel_id"), req, resp)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
