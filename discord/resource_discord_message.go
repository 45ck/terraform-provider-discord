package discord

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restMessageAuthor struct {
	ID string `json:"id"`
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

func resourceDiscordMessage() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMessageCreate,
		ReadContext:   resourceMessageRead,
		UpdateContext: resourceMessageUpdate,
		DeleteContext: resourceMessageDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"channel_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"server_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"author": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"content": {
				AtLeastOneOf: []string{"content", "embed"},
				Type:         schema.TypeString,
				Optional:     true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return old == strings.TrimSuffix(new, "\r\n")
				},
			},
			"timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"edited_timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tts": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"embed": {
				AtLeastOneOf: []string{"content", "embed"},
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				Elem: &schema.Resource{
					Schema: embedSchema(),
				},
			},
			"pinned": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"type": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func setMessageServerID(ctx context.Context, c *RestClient, d *schema.ResourceData, channelID string) {
	var ch restChannelGuild
	if err := c.DoJSON(ctx, "GET", "/channels/"+channelID, nil, nil, &ch); err != nil {
		return
	}
	if ch.GuildID != "" {
		_ = d.Set("server_id", ch.GuildID)
	}
}

func resourceMessageCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	channelID := d.Get("channel_id").(string)
	body := restMessageCreate{
		Content: d.Get("content").(string),
		Tts:     d.Get("tts").(bool),
	}

	if v, ok := d.GetOk("embed"); ok {
		e, err := buildEmbed(v.([]interface{}))
		if err != nil {
			return diag.Errorf("failed to build embed: %s", err.Error())
		}
		body.Embeds = []restEmbed{*e}
	}

	var msg restMessage
	if err := c.DoJSON(ctx, "POST", "/channels/"+channelID+"/messages", nil, body, &msg); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(msg.ID)
	_ = d.Set("type", msg.Type)
	_ = d.Set("timestamp", msg.Timestamp)
	_ = d.Set("author", msg.Author.ID)
	if len(msg.Embeds) > 0 {
		_ = d.Set("embed", unbuildEmbed(&msg.Embeds[0]))
	} else {
		_ = d.Set("embed", nil)
	}
	setMessageServerID(ctx, c, d, channelID)

	if d.Get("pinned").(bool) {
		if err := c.DoJSON(ctx, "PUT", fmt.Sprintf("/channels/%s/pins/%s", channelID, msg.ID), url.Values{}, nil, nil); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceMessageRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	channelID := d.Get("channel_id").(string)
	messageID := d.Id()

	var msg restMessage
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID), nil, nil, &msg); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	setMessageServerID(ctx, c, d, channelID)
	_ = d.Set("type", msg.Type)
	_ = d.Set("tts", msg.Tts)
	_ = d.Set("timestamp", msg.Timestamp)
	_ = d.Set("author", msg.Author.ID)
	_ = d.Set("content", msg.Content)
	_ = d.Set("pinned", msg.Pinned)

	if len(msg.Embeds) > 0 {
		_ = d.Set("embed", unbuildEmbed(&msg.Embeds[0]))
	} else {
		_ = d.Set("embed", nil)
	}
	if msg.EditedTimestamp == "" {
		_ = d.Set("edited_timestamp", nil)
	} else {
		_ = d.Set("edited_timestamp", msg.EditedTimestamp)
	}

	return nil
}

func resourceMessageUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	channelID := d.Get("channel_id").(string)
	messageID := d.Id()

	edit := restMessageEdit{}
	anyEdit := false

	if d.HasChange("content") {
		s := d.Get("content").(string)
		edit.Content = &s
		anyEdit = true
	}
	if d.HasChange("embed") {
		_, n := d.GetChange("embed")
		if len(n.([]interface{})) > 0 {
			e, err := buildEmbed(n.([]interface{}))
			if err != nil {
				return diag.Errorf("failed to build embed: %s", err.Error())
			}
			edit.Embeds = []restEmbed{*e}
		} else {
			// Explicitly clear embeds.
			edit.Embeds = []restEmbed{}
		}
		anyEdit = true
	}

	if anyEdit {
		var msg restMessage
		if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID), nil, edit, &msg); err != nil {
			return diag.FromErr(err)
		}
		if len(msg.Embeds) > 0 {
			_ = d.Set("embed", unbuildEmbed(&msg.Embeds[0]))
		} else {
			_ = d.Set("embed", nil)
		}
		if msg.EditedTimestamp == "" {
			_ = d.Set("edited_timestamp", nil)
		} else {
			_ = d.Set("edited_timestamp", msg.EditedTimestamp)
		}
	}

	if d.HasChange("pinned") {
		if d.Get("pinned").(bool) {
			if err := c.DoJSON(ctx, "PUT", fmt.Sprintf("/channels/%s/pins/%s", channelID, messageID), nil, nil, nil); err != nil {
				return diag.FromErr(err)
			}
		} else {
			if err := c.DoJSON(ctx, "DELETE", fmt.Sprintf("/channels/%s/pins/%s", channelID, messageID), nil, nil, nil); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return resourceMessageRead(ctx, d, m)
}

func resourceMessageDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	channelID := d.Get("channel_id").(string)
	messageID := d.Id()

	if err := c.DoJSON(ctx, "DELETE", fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID), nil, nil, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
