package fw

import (
	"context"
	"fmt"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewWelcomeScreenResource() resource.Resource {
	return &welcomeScreenResource{}
}

type welcomeScreenResource struct {
	c *discord.RestClient
}

type welcomeScreenChannelModel struct {
	ChannelID   types.String `tfsdk:"channel_id"`
	Description types.String `tfsdk:"description"`
	EmojiID     types.String `tfsdk:"emoji_id"`
	EmojiName   types.String `tfsdk:"emoji_name"`
}

type welcomeScreenModel struct {
	ID types.String `tfsdk:"id"`

	ServerID    types.String                `tfsdk:"server_id"`
	Enabled     types.Bool                  `tfsdk:"enabled"`
	Description types.String                `tfsdk:"description"`
	Channel     []welcomeScreenChannelModel `tfsdk:"channel"`
}

type restWelcomeScreen struct {
	Description     string               `json:"description"`
	WelcomeChannels []restWelcomeChannel `json:"welcome_channels"`
	Enabled         bool                 `json:"enabled"`
	GuildID         string               `json:"guild_id"`
}

type restWelcomeChannel struct {
	ChannelID   string `json:"channel_id"`
	Description string `json:"description"`
	EmojiID     string `json:"emoji_id,omitempty"`
	EmojiName   string `json:"emoji_name,omitempty"`
}

func (r *welcomeScreenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_welcome_screen"
}

func (r *welcomeScreenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"enabled": schema.BoolAttribute{
				Optional: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"channel": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"channel_id": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								validate.Snowflake(),
							},
						},
						"description": schema.StringAttribute{Required: true},
						"emoji_id": schema.StringAttribute{
							Optional: true,
							Validators: []validator.String{
								validate.Snowflake(),
							},
						},
						"emoji_name": schema.StringAttribute{Optional: true},
					},
				},
			},
		},
	}
}

func (r *welcomeScreenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *welcomeScreenResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg welcomeScreenModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for i, ch := range cfg.Channel {
		emojiID := ch.EmojiID.ValueString()
		emojiName := ch.EmojiName.ValueString()
		if emojiID != "" && emojiName != "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("channel").AtListIndex(i).AtName("emoji_id"),
				"Invalid configuration",
				"Only one of emoji_id or emoji_name may be set for a welcome screen channel entry.",
			)
		}
	}
}

func expandWelcomeChannels(v []welcomeScreenChannelModel) []restWelcomeChannel {
	out := make([]restWelcomeChannel, 0, len(v))
	for _, raw := range v {
		ch := restWelcomeChannel{
			ChannelID:   raw.ChannelID.ValueString(),
			Description: raw.Description.ValueString(),
		}
		if s := raw.EmojiID.ValueString(); s != "" {
			ch.EmojiID = s
		}
		if s := raw.EmojiName.ValueString(); s != "" {
			ch.EmojiName = s
		}
		out = append(out, ch)
	}
	return out
}

func flattenWelcomeChannels(v []restWelcomeChannel) []welcomeScreenChannelModel {
	out := make([]welcomeScreenChannelModel, 0, len(v))
	for _, ch := range v {
		out = append(out, welcomeScreenChannelModel{
			ChannelID:   types.StringValue(ch.ChannelID),
			Description: types.StringValue(ch.Description),
			EmojiID:     types.StringValue(ch.EmojiID),
			EmojiName:   types.StringValue(ch.EmojiName),
		})
	}
	return out
}

func (r *welcomeScreenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan welcomeScreenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.upsert(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *welcomeScreenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan welcomeScreenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.upsert(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *welcomeScreenResource) upsert(ctx context.Context, plan *welcomeScreenModel, diags discordFrameworkDiagnostics) {
	serverID := plan.ServerID.ValueString()

	body := map[string]any{
		"enabled":     !plan.Enabled.IsNull() && plan.Enabled.ValueBool(),
		"description": plan.Description.ValueString(),
	}
	if plan.Channel != nil {
		body["welcome_channels"] = expandWelcomeChannels(plan.Channel)
	} else {
		body["welcome_channels"] = []restWelcomeChannel{}
	}

	var out restWelcomeScreen
	if err := r.c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s/welcome-screen", serverID), nil, body, &out); err != nil {
		diags.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(serverID)
	r.readIntoState(ctx, plan, diags)
}

func (r *welcomeScreenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state welcomeScreenModel
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

func (r *welcomeScreenResource) readIntoState(ctx context.Context, state *welcomeScreenModel, diags discordFrameworkDiagnostics) {
	serverID := state.ID.ValueString()
	if serverID == "" {
		serverID = state.ServerID.ValueString()
	}

	var out restWelcomeScreen
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/welcome-screen", serverID), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	state.ID = types.StringValue(serverID)
	state.ServerID = types.StringValue(serverID)
	state.Enabled = types.BoolValue(out.Enabled)
	state.Description = types.StringValue(out.Description)
	state.Channel = flattenWelcomeChannels(out.WelcomeChannels)
}

func (r *welcomeScreenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state welcomeScreenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := state.ID.ValueString()
	body := map[string]any{
		"enabled":          false,
		"description":      "",
		"welcome_channels": []restWelcomeChannel{},
	}
	if err := r.c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s/welcome-screen", serverID), nil, body, nil); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *welcomeScreenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("server_id"), req, resp)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
