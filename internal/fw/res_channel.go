package fw

import (
	"context"
	"fmt"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw/fwutil"
	"github.com/aequasi/discord-terraform/internal/fw/planmod"
	"github.com/aequasi/discord-terraform/internal/fw/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewChannelResource() resource.Resource {
	return &channelResource{}
}

type channelResource struct {
	c *discord.RestClient
}

type channelForumTagModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Moderated types.Bool   `tfsdk:"moderated"`
	EmojiID   types.String `tfsdk:"emoji_id"`
	EmojiName types.String `tfsdk:"emoji_name"`
}

type channelDefaultReactionModel struct {
	EmojiID   types.String `tfsdk:"emoji_id"`
	EmojiName types.String `tfsdk:"emoji_name"`
}

type channelResourceModel struct {
	ID types.String `tfsdk:"id"`

	ServerID types.String `tfsdk:"server_id"`
	Type     types.String `tfsdk:"type"`
	Name     types.String `tfsdk:"name"`
	Reason   types.String `tfsdk:"reason"`

	Position types.Int64  `tfsdk:"position"`
	ParentID types.String `tfsdk:"parent_id"`
	Topic    types.String `tfsdk:"topic"`
	NSFW     types.Bool   `tfsdk:"nsfw"`

	RateLimitPerUser              types.Int64  `tfsdk:"rate_limit_per_user"`
	Bitrate                       types.Int64  `tfsdk:"bitrate"`
	UserLimit                     types.Int64  `tfsdk:"user_limit"`
	RTCRegion                     types.String `tfsdk:"rtc_region"`
	VideoQualityMode              types.Int64  `tfsdk:"video_quality_mode"`
	DefaultAutoArchiveDuration    types.Int64  `tfsdk:"default_auto_archive_duration"`
	DefaultThreadRateLimitPerUser types.Int64  `tfsdk:"default_thread_rate_limit_per_user"`

	AvailableTag         []channelForumTagModel       `tfsdk:"available_tag"`
	DefaultReactionEmoji *channelDefaultReactionModel `tfsdk:"default_reaction_emoji"`
	DefaultSortOrder     types.Int64                  `tfsdk:"default_sort_order"`
	DefaultForumLayout   types.Int64                  `tfsdk:"default_forum_layout"`
}

type restChannel struct {
	ID                     string               `json:"id"`
	GuildID                string               `json:"guild_id"`
	Name                   string               `json:"name"`
	Type                   uint                 `json:"type"`
	Position               int                  `json:"position"`
	ParentID               string               `json:"parent_id"`
	Topic                  string               `json:"topic"`
	NSFW                   bool                 `json:"nsfw"`
	RateLimitPerUser       int                  `json:"rate_limit_per_user"`
	Bitrate                int                  `json:"bitrate"`
	UserLimit              int                  `json:"user_limit"`
	RTCRegion              string               `json:"rtc_region"`
	VideoQualityMode       int                  `json:"video_quality_mode"`
	DefaultAutoArchiveDur  int                  `json:"default_auto_archive_duration"`
	DefaultThreadRateLimit int                  `json:"default_thread_rate_limit_per_user"`
	AvailableTags          []restForumTag       `json:"available_tags"`
	DefaultReactionEmoji   *restDefaultReaction `json:"default_reaction_emoji"`
	DefaultSortOrder       int                  `json:"default_sort_order"`
	DefaultForumLayout     int                  `json:"default_forum_layout"`
}

type restForumTag struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Moderated bool   `json:"moderated"`
	EmojiID   string `json:"emoji_id"`
	EmojiName string `json:"emoji_name"`
}

type restDefaultReaction struct {
	EmojiID   string `json:"emoji_id"`
	EmojiName string `json:"emoji_name"`
}

func (r *channelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel"
}

func (r *channelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"type": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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

			"position": schema.Int64Attribute{Optional: true},
			"parent_id": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"topic": schema.StringAttribute{Optional: true},
			"nsfw":  schema.BoolAttribute{Optional: true},

			"rate_limit_per_user": schema.Int64Attribute{Optional: true},
			"bitrate":             schema.Int64Attribute{Optional: true},
			"user_limit":          schema.Int64Attribute{Optional: true},
			"rtc_region":          schema.StringAttribute{Optional: true},
			"video_quality_mode":  schema.Int64Attribute{Optional: true},
			"default_auto_archive_duration": schema.Int64Attribute{
				Optional: true,
			},
			"default_thread_rate_limit_per_user": schema.Int64Attribute{
				Optional: true,
			},

			"available_tag": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.StringAttribute{Optional: true},
						"name":       schema.StringAttribute{Required: true},
						"moderated":  schema.BoolAttribute{Optional: true},
						"emoji_id":   schema.StringAttribute{Optional: true},
						"emoji_name": schema.StringAttribute{Optional: true},
					},
				},
				Description: "Forum/media available tags.",
			},
			"default_reaction_emoji": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"emoji_id":   schema.StringAttribute{Optional: true},
					"emoji_name": schema.StringAttribute{Optional: true},
				},
				Description: "Forum default reaction emoji.",
			},
			"default_sort_order": schema.Int64Attribute{
				Optional:    true,
				Description: "Forum default sort order.",
			},
			"default_forum_layout": schema.Int64Attribute{
				Optional:    true,
				Description: "Forum default layout.",
			},
		},
	}
}

func (r *channelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *channelResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg channelResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for i, tag := range cfg.AvailableTag {
		emojiID := tag.EmojiID.ValueString()
		emojiName := tag.EmojiName.ValueString()
		if emojiID != "" && emojiName != "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("available_tag").AtListIndex(i).AtName("emoji_id"),
				"Invalid configuration",
				"Only one of emoji_id or emoji_name may be set for a forum tag.",
			)
		}
	}

	if cfg.DefaultReactionEmoji != nil {
		emojiID := cfg.DefaultReactionEmoji.EmojiID.ValueString()
		emojiName := cfg.DefaultReactionEmoji.EmojiName.ValueString()
		if emojiID != "" && emojiName != "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("default_reaction_emoji").AtName("emoji_id"),
				"Invalid configuration",
				"Only one of emoji_id or emoji_name may be set for default_reaction_emoji.",
			)
		}
	}
}

func validateChannelType(t string) (uint, error) {
	v, ok := discord.GetDiscordChannelType(t)
	if !ok {
		return 0, fmt.Errorf("unsupported channel type: %s", t)
	}
	switch t {
	case "announcement_thread", "public_thread", "private_thread":
		return 0, fmt.Errorf("thread types are not created via discord_channel; use discord_thread instead")
	}
	return v, nil
}

func expandForumTags(v []channelForumTagModel) []restForumTag {
	out := make([]restForumTag, 0, len(v))
	for _, raw := range v {
		tag := restForumTag{
			ID:        raw.ID.ValueString(),
			Name:      raw.Name.ValueString(),
			Moderated: !raw.Moderated.IsNull() && raw.Moderated.ValueBool(),
		}
		if !raw.EmojiID.IsNull() && raw.EmojiID.ValueString() != "" {
			tag.EmojiID = raw.EmojiID.ValueString()
		}
		if !raw.EmojiName.IsNull() && raw.EmojiName.ValueString() != "" {
			tag.EmojiName = raw.EmojiName.ValueString()
		}
		out = append(out, tag)
	}
	return out
}

func flattenForumTags(v []restForumTag) []channelForumTagModel {
	out := make([]channelForumTagModel, 0, len(v))
	for _, tag := range v {
		out = append(out, channelForumTagModel{
			ID:        types.StringValue(tag.ID),
			Name:      types.StringValue(tag.Name),
			Moderated: types.BoolValue(tag.Moderated),
			EmojiID:   types.StringValue(tag.EmojiID),
			EmojiName: types.StringValue(tag.EmojiName),
		})
	}
	return out
}

func equalForumTags(a, b []channelForumTagModel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if fwutil.ChangedString(a[i].ID, b[i].ID) ||
			fwutil.ChangedString(a[i].Name, b[i].Name) ||
			fwutil.ChangedBool(a[i].Moderated, b[i].Moderated) ||
			fwutil.ChangedString(a[i].EmojiID, b[i].EmojiID) ||
			fwutil.ChangedString(a[i].EmojiName, b[i].EmojiName) {
			return false
		}
	}
	return true
}

func equalDefaultReaction(a, b *channelDefaultReactionModel) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return !fwutil.ChangedString(a.EmojiID, b.EmojiID) && !fwutil.ChangedString(a.EmojiName, b.EmojiName)
}

func (r *channelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan channelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	typ, err := validateChannelType(plan.Type.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid type", err.Error())
		return
	}

	body := map[string]any{
		"type": int(typ),
		"name": plan.Name.ValueString(),
	}

	if !plan.Position.IsNull() {
		body["position"] = int(plan.Position.ValueInt64())
	}
	if !plan.Topic.IsNull() && plan.Topic.ValueString() != "" {
		body["topic"] = plan.Topic.ValueString()
	}
	if !plan.NSFW.IsNull() {
		body["nsfw"] = plan.NSFW.ValueBool()
	}
	if !plan.RateLimitPerUser.IsNull() {
		body["rate_limit_per_user"] = int(plan.RateLimitPerUser.ValueInt64())
	}
	if !plan.Bitrate.IsNull() {
		body["bitrate"] = int(plan.Bitrate.ValueInt64())
	}
	if !plan.UserLimit.IsNull() {
		body["user_limit"] = int(plan.UserLimit.ValueInt64())
	}
	if !plan.RTCRegion.IsNull() && plan.RTCRegion.ValueString() != "" {
		body["rtc_region"] = plan.RTCRegion.ValueString()
	}
	if !plan.VideoQualityMode.IsNull() {
		body["video_quality_mode"] = int(plan.VideoQualityMode.ValueInt64())
	}
	if !plan.DefaultAutoArchiveDuration.IsNull() {
		body["default_auto_archive_duration"] = int(plan.DefaultAutoArchiveDuration.ValueInt64())
	}
	if !plan.DefaultThreadRateLimitPerUser.IsNull() {
		body["default_thread_rate_limit_per_user"] = int(plan.DefaultThreadRateLimitPerUser.ValueInt64())
	}
	if plan.AvailableTag != nil {
		body["available_tags"] = expandForumTags(plan.AvailableTag)
	}
	if plan.DefaultReactionEmoji != nil {
		body["default_reaction_emoji"] = map[string]any{
			"emoji_id":   plan.DefaultReactionEmoji.EmojiID.ValueString(),
			"emoji_name": plan.DefaultReactionEmoji.EmojiName.ValueString(),
		}
	}
	if !plan.DefaultSortOrder.IsNull() {
		body["default_sort_order"] = int(plan.DefaultSortOrder.ValueInt64())
	}
	if !plan.DefaultForumLayout.IsNull() {
		body["default_forum_layout"] = int(plan.DefaultForumLayout.ValueInt64())
	}
	if !plan.ParentID.IsNull() && plan.ParentID.ValueString() != "" {
		body["parent_id"] = plan.ParentID.ValueString()
	}

	var out restChannel
	if err := r.c.DoJSONWithReason(ctx, "POST", fmt.Sprintf("/guilds/%s/channels", plan.ServerID.ValueString()), nil, body, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(out.ID)
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *channelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state channelResourceModel
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

func (r *channelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan channelResourceModel
	var state channelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	if fwutil.ChangedString(plan.Name, state.Name) {
		body["name"] = plan.Name.ValueString()
	}
	if fwutil.ChangedInt64(plan.Position, state.Position) {
		if plan.Position.IsNull() {
			body["position"] = 0
		} else {
			body["position"] = int(plan.Position.ValueInt64())
		}
	}
	if fwutil.ChangedString(plan.ParentID, state.ParentID) {
		if plan.ParentID.IsNull() || plan.ParentID.ValueString() == "" {
			body["parent_id"] = nil
		} else {
			body["parent_id"] = plan.ParentID.ValueString()
		}
	}
	if fwutil.ChangedString(plan.Topic, state.Topic) {
		if plan.Topic.IsNull() {
			body["topic"] = ""
		} else {
			body["topic"] = plan.Topic.ValueString()
		}
	}
	if fwutil.ChangedBool(plan.NSFW, state.NSFW) {
		body["nsfw"] = !plan.NSFW.IsNull() && plan.NSFW.ValueBool()
	}
	if fwutil.ChangedInt64(plan.RateLimitPerUser, state.RateLimitPerUser) {
		body["rate_limit_per_user"] = int(plan.RateLimitPerUser.ValueInt64())
	}
	if fwutil.ChangedInt64(plan.Bitrate, state.Bitrate) {
		body["bitrate"] = int(plan.Bitrate.ValueInt64())
	}
	if fwutil.ChangedInt64(plan.UserLimit, state.UserLimit) {
		body["user_limit"] = int(plan.UserLimit.ValueInt64())
	}
	if fwutil.ChangedString(plan.RTCRegion, state.RTCRegion) {
		if plan.RTCRegion.IsNull() || plan.RTCRegion.ValueString() == "" {
			body["rtc_region"] = nil
		} else {
			body["rtc_region"] = plan.RTCRegion.ValueString()
		}
	}
	if fwutil.ChangedInt64(plan.VideoQualityMode, state.VideoQualityMode) {
		body["video_quality_mode"] = int(plan.VideoQualityMode.ValueInt64())
	}
	if fwutil.ChangedInt64(plan.DefaultAutoArchiveDuration, state.DefaultAutoArchiveDuration) {
		body["default_auto_archive_duration"] = int(plan.DefaultAutoArchiveDuration.ValueInt64())
	}
	if fwutil.ChangedInt64(plan.DefaultThreadRateLimitPerUser, state.DefaultThreadRateLimitPerUser) {
		body["default_thread_rate_limit_per_user"] = int(plan.DefaultThreadRateLimitPerUser.ValueInt64())
	}

	if !equalForumTags(plan.AvailableTag, state.AvailableTag) {
		body["available_tags"] = expandForumTags(plan.AvailableTag)
	}
	if !equalDefaultReaction(plan.DefaultReactionEmoji, state.DefaultReactionEmoji) {
		if plan.DefaultReactionEmoji == nil {
			body["default_reaction_emoji"] = nil
		} else {
			body["default_reaction_emoji"] = map[string]any{
				"emoji_id":   plan.DefaultReactionEmoji.EmojiID.ValueString(),
				"emoji_name": plan.DefaultReactionEmoji.EmojiName.ValueString(),
			}
		}
	}
	if fwutil.ChangedInt64(plan.DefaultSortOrder, state.DefaultSortOrder) {
		body["default_sort_order"] = int(plan.DefaultSortOrder.ValueInt64())
	}
	if fwutil.ChangedInt64(plan.DefaultForumLayout, state.DefaultForumLayout) {
		body["default_forum_layout"] = int(plan.DefaultForumLayout.ValueInt64())
	}

	if len(body) == 0 {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	var out restChannel
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/channels/%s", state.ID.ValueString()), nil, body, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = state.ID
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *channelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state channelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/channels/%s", state.ID.ValueString()), nil, nil, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *channelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *channelResource) readIntoState(ctx context.Context, state *channelResourceModel, diags discordFrameworkDiagnostics) {
	var out restChannel
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/channels/%s", state.ID.ValueString()), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	t, ok := discord.GetTextChannelType(out.Type)
	if ok {
		state.Type = types.StringValue(t)
	} else {
		state.Type = types.StringValue(fmt.Sprintf("%d", out.Type))
	}

	state.ServerID = types.StringValue(out.GuildID)
	state.Name = types.StringValue(out.Name)
	state.Position = types.Int64Value(int64(out.Position))
	if out.ParentID != "" {
		state.ParentID = types.StringValue(out.ParentID)
	} else {
		state.ParentID = types.StringNull()
	}
	state.Topic = types.StringValue(out.Topic)
	state.NSFW = types.BoolValue(out.NSFW)
	state.RateLimitPerUser = types.Int64Value(int64(out.RateLimitPerUser))
	state.Bitrate = types.Int64Value(int64(out.Bitrate))
	state.UserLimit = types.Int64Value(int64(out.UserLimit))
	state.RTCRegion = types.StringValue(out.RTCRegion)
	state.VideoQualityMode = types.Int64Value(int64(out.VideoQualityMode))
	state.DefaultAutoArchiveDuration = types.Int64Value(int64(out.DefaultAutoArchiveDur))
	state.DefaultThreadRateLimitPerUser = types.Int64Value(int64(out.DefaultThreadRateLimit))

	if out.AvailableTags != nil {
		state.AvailableTag = flattenForumTags(out.AvailableTags)
	} else {
		state.AvailableTag = nil
	}

	if out.DefaultReactionEmoji != nil {
		state.DefaultReactionEmoji = &channelDefaultReactionModel{
			EmojiID:   types.StringValue(out.DefaultReactionEmoji.EmojiID),
			EmojiName: types.StringValue(out.DefaultReactionEmoji.EmojiName),
		}
	} else {
		state.DefaultReactionEmoji = nil
	}

	state.DefaultSortOrder = types.Int64Value(int64(out.DefaultSortOrder))
	state.DefaultForumLayout = types.Int64Value(int64(out.DefaultForumLayout))
}
