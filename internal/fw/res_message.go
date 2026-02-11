package fw

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/aequasi/discord-terraform/discord"
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

func NewMessageResource() resource.Resource {
	return &messageResource{}
}

type messageResource struct {
	c *discord.RestClient
}

type messageEmbedFooterModel struct {
	Text    types.String `tfsdk:"text"`
	IconURL types.String `tfsdk:"icon_url"`
}

type messageEmbedImageModel struct {
	URL      types.String `tfsdk:"url"`
	ProxyURL types.String `tfsdk:"proxy_url"`
	Height   types.Int64  `tfsdk:"height"`
	Width    types.Int64  `tfsdk:"width"`
}

type messageEmbedThumbnailModel struct {
	URL      types.String `tfsdk:"url"`
	ProxyURL types.String `tfsdk:"proxy_url"`
	Height   types.Int64  `tfsdk:"height"`
	Width    types.Int64  `tfsdk:"width"`
}

type messageEmbedVideoModel struct {
	URL    types.String `tfsdk:"url"`
	Height types.Int64  `tfsdk:"height"`
	Width  types.Int64  `tfsdk:"width"`
}

type messageEmbedProviderModel struct {
	Name types.String `tfsdk:"name"`
	URL  types.String `tfsdk:"url"`
}

type messageEmbedAuthorModel struct {
	Name         types.String `tfsdk:"name"`
	URL          types.String `tfsdk:"url"`
	IconURL      types.String `tfsdk:"icon_url"`
	ProxyIconURL types.String `tfsdk:"proxy_icon_url"`
}

type messageEmbedFieldModel struct {
	Name   types.String `tfsdk:"name"`
	Value  types.String `tfsdk:"value"`
	Inline types.Bool   `tfsdk:"inline"`
}

type messageEmbedModel struct {
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	URL         types.String `tfsdk:"url"`
	Timestamp   types.String `tfsdk:"timestamp"`
	Color       types.Int64  `tfsdk:"color"`

	Footer    *messageEmbedFooterModel    `tfsdk:"footer"`
	Image     *messageEmbedImageModel     `tfsdk:"image"`
	Thumbnail *messageEmbedThumbnailModel `tfsdk:"thumbnail"`
	Video     *messageEmbedVideoModel     `tfsdk:"video"`
	Provider  *messageEmbedProviderModel  `tfsdk:"provider"`
	Author    *messageEmbedAuthorModel    `tfsdk:"author"`
	Fields    []messageEmbedFieldModel    `tfsdk:"fields"`
}

type messageModel struct {
	ID types.String `tfsdk:"id"`

	ChannelID types.String `tfsdk:"channel_id"`

	ServerID types.String `tfsdk:"server_id"`
	Author   types.String `tfsdk:"author"`

	Content         types.String `tfsdk:"content"`
	Timestamp       types.String `tfsdk:"timestamp"`
	EditedTimestamp types.String `tfsdk:"edited_timestamp"`

	TTS    types.Bool         `tfsdk:"tts"`
	Embed  *messageEmbedModel `tfsdk:"embed"`
	Pinned types.Bool         `tfsdk:"pinned"`

	Type types.Int64 `tfsdk:"type"`
}

type restMessageAuthor struct {
	ID string `json:"id"`
}

type restEmbed struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
	Color       int    `json:"color,omitempty"`

	Footer    *restEmbedFooter    `json:"footer,omitempty"`
	Image     *restEmbedImage     `json:"image,omitempty"`
	Thumbnail *restEmbedThumbnail `json:"thumbnail,omitempty"`
	Video     *restEmbedVideo     `json:"video,omitempty"`
	Provider  *restEmbedProvider  `json:"provider,omitempty"`
	Author    *restEmbedAuthor    `json:"author,omitempty"`
	Fields    []restEmbedField    `json:"fields,omitempty"`
}

type restEmbedFooter struct {
	Text    string `json:"text,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

type restEmbedImage struct {
	URL      string `json:"url,omitempty"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Height   int    `json:"height,omitempty"`
	Width    int    `json:"width,omitempty"`
}

type restEmbedThumbnail struct {
	URL      string `json:"url,omitempty"`
	ProxyURL string `json:"proxy_url,omitempty"`
	Height   int    `json:"height,omitempty"`
	Width    int    `json:"width,omitempty"`
}

type restEmbedVideo struct {
	URL    string `json:"url,omitempty"`
	Height int    `json:"height,omitempty"`
	Width  int    `json:"width,omitempty"`
}

type restEmbedProvider struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type restEmbedAuthor struct {
	Name         string `json:"name,omitempty"`
	URL          string `json:"url,omitempty"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

type restEmbedField struct {
	Name   string `json:"name,omitempty"`
	Value  string `json:"value,omitempty"`
	Inline bool   `json:"inline,omitempty"`
}

type restMessage struct {
	ID              string            `json:"id"`
	ChannelID       string            `json:"channel_id"`
	Content         string            `json:"content"`
	Tts             bool              `json:"tts"`
	Pinned          bool              `json:"pinned"`
	Type            int               `json:"type"`
	Timestamp       string            `json:"timestamp"`
	EditedTimestamp string            `json:"edited_timestamp"`
	Author          restMessageAuthor `json:"author"`
	Embeds          []restEmbed       `json:"embeds"`
}

type restChannelGuild struct {
	ID      string `json:"id"`
	GuildID string `json:"guild_id"`
}

type restMessageCreate struct {
	Content string      `json:"content,omitempty"`
	Tts     bool        `json:"tts,omitempty"`
	Embeds  []restEmbed `json:"embeds,omitempty"`
}

type restMessageEdit struct {
	Content *string     `json:"content,omitempty"`
	Embeds  []restEmbed `json:"embeds,omitempty"`
}

func (r *messageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_message"
}

func (r *messageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},

			"channel_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"server_id": schema.StringAttribute{Computed: true},
			"author":    schema.StringAttribute{Computed: true},

			"content": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					planmod.TrimTrailingCRLFString(),
				},
			},
			"timestamp":        schema.StringAttribute{Computed: true},
			"edited_timestamp": schema.StringAttribute{Computed: true},

			"tts": schema.BoolAttribute{
				Optional: true,
			},
			"embed": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"title":       schema.StringAttribute{Optional: true},
					"description": schema.StringAttribute{Optional: true},
					"url":         schema.StringAttribute{Optional: true},
					"timestamp": schema.StringAttribute{
						Optional: true,
						Validators: []validator.String{
							validate.RFC3339Timestamp(),
						},
					},
					"color": schema.Int64Attribute{Optional: true},
					"footer": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"text":     schema.StringAttribute{Required: true},
							"icon_url": schema.StringAttribute{Optional: true},
						},
					},
					"image": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"url":       schema.StringAttribute{Required: true},
							"proxy_url": schema.StringAttribute{Computed: true},
							"height":    schema.Int64Attribute{Optional: true},
							"width":     schema.Int64Attribute{Optional: true},
						},
					},
					"thumbnail": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"url":       schema.StringAttribute{Required: true},
							"proxy_url": schema.StringAttribute{Computed: true},
							"height":    schema.Int64Attribute{Optional: true},
							"width":     schema.Int64Attribute{Optional: true},
						},
					},
					"video": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"url":    schema.StringAttribute{Required: true},
							"height": schema.Int64Attribute{Optional: true},
							"width":  schema.Int64Attribute{Optional: true},
						},
					},
					"provider": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{Optional: true},
							"url":  schema.StringAttribute{Optional: true},
						},
					},
					"author": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"name":           schema.StringAttribute{Optional: true},
							"url":            schema.StringAttribute{Optional: true},
							"icon_url":       schema.StringAttribute{Optional: true},
							"proxy_icon_url": schema.StringAttribute{Computed: true},
						},
					},
					"fields": schema.ListNestedAttribute{
						Optional: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name":   schema.StringAttribute{Required: true},
								"value":  schema.StringAttribute{Optional: true},
								"inline": schema.BoolAttribute{Optional: true},
							},
						},
					},
				},
			},
			"pinned": schema.BoolAttribute{
				Optional: true,
			},
			"type": schema.Int64Attribute{Computed: true},
		},
	}
}

func (r *messageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func embedToRest(m *messageEmbedModel) restEmbed {
	if m == nil {
		return restEmbed{}
	}
	e := restEmbed{
		Title:       m.Title.ValueString(),
		Description: m.Description.ValueString(),
		URL:         m.URL.ValueString(),
		Timestamp:   m.Timestamp.ValueString(),
		Color:       int(m.Color.ValueInt64()),
	}
	if m.Footer != nil {
		e.Footer = &restEmbedFooter{
			Text:    m.Footer.Text.ValueString(),
			IconURL: m.Footer.IconURL.ValueString(),
		}
	}
	if m.Image != nil {
		e.Image = &restEmbedImage{
			URL:    m.Image.URL.ValueString(),
			Height: int(m.Image.Height.ValueInt64()),
			Width:  int(m.Image.Width.ValueInt64()),
		}
	}
	if m.Thumbnail != nil {
		e.Thumbnail = &restEmbedThumbnail{
			URL:    m.Thumbnail.URL.ValueString(),
			Height: int(m.Thumbnail.Height.ValueInt64()),
			Width:  int(m.Thumbnail.Width.ValueInt64()),
		}
	}
	if m.Video != nil {
		e.Video = &restEmbedVideo{
			URL:    m.Video.URL.ValueString(),
			Height: int(m.Video.Height.ValueInt64()),
			Width:  int(m.Video.Width.ValueInt64()),
		}
	}
	if m.Provider != nil {
		e.Provider = &restEmbedProvider{
			Name: m.Provider.Name.ValueString(),
			URL:  m.Provider.URL.ValueString(),
		}
	}
	if m.Author != nil {
		e.Author = &restEmbedAuthor{
			Name:    m.Author.Name.ValueString(),
			URL:     m.Author.URL.ValueString(),
			IconURL: m.Author.IconURL.ValueString(),
		}
	}
	if m.Fields != nil {
		fields := make([]restEmbedField, 0, len(m.Fields))
		for _, f := range m.Fields {
			fields = append(fields, restEmbedField{
				Name:   f.Name.ValueString(),
				Value:  f.Value.ValueString(),
				Inline: !f.Inline.IsNull() && f.Inline.ValueBool(),
			})
		}
		e.Fields = fields
	}
	return e
}

func restToEmbed(in *restEmbed) *messageEmbedModel {
	if in == nil {
		return nil
	}
	out := &messageEmbedModel{
		Title:       types.StringValue(in.Title),
		Description: types.StringValue(in.Description),
		URL:         types.StringValue(in.URL),
		Timestamp:   types.StringValue(in.Timestamp),
		Color:       types.Int64Value(int64(in.Color)),
	}
	if in.Footer != nil {
		out.Footer = &messageEmbedFooterModel{
			Text:    types.StringValue(in.Footer.Text),
			IconURL: types.StringValue(in.Footer.IconURL),
		}
	}
	if in.Image != nil {
		out.Image = &messageEmbedImageModel{
			URL:      types.StringValue(in.Image.URL),
			ProxyURL: types.StringValue(in.Image.ProxyURL),
			Height:   types.Int64Value(int64(in.Image.Height)),
			Width:    types.Int64Value(int64(in.Image.Width)),
		}
	}
	if in.Thumbnail != nil {
		out.Thumbnail = &messageEmbedThumbnailModel{
			URL:      types.StringValue(in.Thumbnail.URL),
			ProxyURL: types.StringValue(in.Thumbnail.ProxyURL),
			Height:   types.Int64Value(int64(in.Thumbnail.Height)),
			Width:    types.Int64Value(int64(in.Thumbnail.Width)),
		}
	}
	if in.Video != nil {
		out.Video = &messageEmbedVideoModel{
			URL:    types.StringValue(in.Video.URL),
			Height: types.Int64Value(int64(in.Video.Height)),
			Width:  types.Int64Value(int64(in.Video.Width)),
		}
	}
	if in.Provider != nil {
		out.Provider = &messageEmbedProviderModel{
			Name: types.StringValue(in.Provider.Name),
			URL:  types.StringValue(in.Provider.URL),
		}
	}
	if in.Author != nil {
		out.Author = &messageEmbedAuthorModel{
			Name:         types.StringValue(in.Author.Name),
			URL:          types.StringValue(in.Author.URL),
			IconURL:      types.StringValue(in.Author.IconURL),
			ProxyIconURL: types.StringValue(in.Author.ProxyIconURL),
		}
	}
	if in.Fields != nil {
		fields := make([]messageEmbedFieldModel, 0, len(in.Fields))
		for _, f := range in.Fields {
			fields = append(fields, messageEmbedFieldModel{
				Name:   types.StringValue(f.Name),
				Value:  types.StringValue(f.Value),
				Inline: types.BoolValue(f.Inline),
			})
		}
		out.Fields = fields
	}
	return out
}

func (r *messageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan messageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channelID := plan.ChannelID.ValueString()
	content := ""
	if !plan.Content.IsNull() {
		content = plan.Content.ValueString()
	}
	if content == "" && plan.Embed == nil {
		resp.Diagnostics.AddError("Invalid configuration", "at least one of content or embed must be set")
		return
	}

	body := restMessageCreate{
		Content: content,
		Tts:     !plan.TTS.IsNull() && plan.TTS.ValueBool(),
	}
	if plan.Embed != nil {
		body.Embeds = []restEmbed{embedToRest(plan.Embed)}
	}

	var msg restMessage
	if err := r.c.DoJSON(ctx, "POST", "/channels/"+channelID+"/messages", nil, body, &msg); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(msg.ID)
	plan.Type = types.Int64Value(int64(msg.Type))
	plan.Timestamp = types.StringValue(msg.Timestamp)
	plan.Author = types.StringValue(msg.Author.ID)
	if len(msg.Embeds) > 0 {
		plan.Embed = restToEmbed(&msg.Embeds[0])
	} else {
		plan.Embed = nil
	}

	r.setMessageServerID(ctx, &plan, channelID)

	if !plan.Pinned.IsNull() && plan.Pinned.ValueBool() {
		if err := r.c.DoJSON(ctx, "PUT", fmt.Sprintf("/channels/%s/pins/%s", channelID, msg.ID), url.Values{}, nil, nil); err != nil {
			resp.Diagnostics.AddError("Discord API error", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *messageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state messageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channelID := state.ChannelID.ValueString()
	messageID := state.ID.ValueString()

	var msg restMessage
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID), nil, nil, &msg); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	state.Type = types.Int64Value(int64(msg.Type))
	state.TTS = types.BoolValue(msg.Tts)
	state.Timestamp = types.StringValue(msg.Timestamp)
	state.Author = types.StringValue(msg.Author.ID)
	state.Content = types.StringValue(msg.Content)
	state.Pinned = types.BoolValue(msg.Pinned)

	if len(msg.Embeds) > 0 {
		state.Embed = restToEmbed(&msg.Embeds[0])
	} else {
		state.Embed = nil
	}

	if msg.EditedTimestamp == "" {
		state.EditedTimestamp = types.StringNull()
	} else {
		state.EditedTimestamp = types.StringValue(msg.EditedTimestamp)
	}

	r.setMessageServerID(ctx, &state, channelID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *messageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan messageModel
	var state messageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channelID := state.ChannelID.ValueString()
	messageID := state.ID.ValueString()

	edit := restMessageEdit{}
	anyEdit := false

	if !plan.Content.Equal(state.Content) {
		s := ""
		if !plan.Content.IsNull() {
			s = plan.Content.ValueString()
		}
		edit.Content = &s
		anyEdit = true
	}
	if (plan.Embed == nil) != (state.Embed == nil) || (plan.Embed != nil && state.Embed != nil && !plan.Embed.Title.Equal(state.Embed.Title)) {
		// Conservative: if embed presence changed, or title changed, send embed set/clear.
		// This avoids implementing deep equality across embed fields.
		if plan.Embed != nil {
			edit.Embeds = []restEmbed{embedToRest(plan.Embed)}
		} else {
			edit.Embeds = []restEmbed{}
		}
		anyEdit = true
	}

	if anyEdit {
		var msg restMessage
		if err := r.c.DoJSON(ctx, "PATCH", fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID), nil, edit, &msg); err != nil {
			resp.Diagnostics.AddError("Discord API error", err.Error())
			return
		}
		if len(msg.Embeds) > 0 {
			plan.Embed = restToEmbed(&msg.Embeds[0])
		} else {
			plan.Embed = nil
		}
		if msg.EditedTimestamp == "" {
			plan.EditedTimestamp = types.StringNull()
		} else {
			plan.EditedTimestamp = types.StringValue(msg.EditedTimestamp)
		}
	}

	if !plan.Pinned.Equal(state.Pinned) {
		if !plan.Pinned.IsNull() && plan.Pinned.ValueBool() {
			if err := r.c.DoJSON(ctx, "PUT", fmt.Sprintf("/channels/%s/pins/%s", channelID, messageID), nil, nil, nil); err != nil {
				resp.Diagnostics.AddError("Discord API error", err.Error())
				return
			}
		} else {
			if err := r.c.DoJSON(ctx, "DELETE", fmt.Sprintf("/channels/%s/pins/%s", channelID, messageID), nil, nil, nil); err != nil {
				resp.Diagnostics.AddError("Discord API error", err.Error())
				return
			}
		}
	}

	plan.ID = state.ID
	plan.ChannelID = state.ChannelID
	r.setMessageServerID(ctx, &plan, channelID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *messageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state messageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channelID := state.ChannelID.ValueString()
	messageID := state.ID.ValueString()

	if err := r.c.DoJSON(ctx, "DELETE", fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID), nil, nil, nil); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *messageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: message_id, but channel_id is required for API operations, so require composite.
	// Accept channel_id:message_id.
	ch, mid, err := parseTwoIDs(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected channel_id:message_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("channel_id"), ch)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), mid)...)
}

func parseTwoIDs(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected attribute1:attribute2", id)
	}
	return parts[0], parts[1], nil
}

func (r *messageResource) setMessageServerID(ctx context.Context, state *messageModel, channelID string) {
	var ch restChannelGuild
	if err := r.c.DoJSON(ctx, "GET", "/channels/"+channelID, nil, nil, &ch); err != nil {
		return
	}
	if ch.GuildID != "" {
		state.ServerID = types.StringValue(ch.GuildID)
	}
}
