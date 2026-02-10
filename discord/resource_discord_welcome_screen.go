package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

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

func resourceDiscordWelcomeScreen() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordWelcomeScreenUpsert,
		ReadContext:   resourceDiscordWelcomeScreenRead,
		UpdateContext: resourceDiscordWelcomeScreenUpsert,
		DeleteContext: resourceDiscordWelcomeScreenDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"channel": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"channel_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"description": {
							Type:     schema.TypeString,
							Required: true,
						},
						"emoji_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"emoji_name": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func expandWelcomeChannels(v []interface{}) []restWelcomeChannel {
	out := make([]restWelcomeChannel, 0, len(v))
	for _, raw := range v {
		m := raw.(map[string]interface{})
		ch := restWelcomeChannel{
			ChannelID:   m["channel_id"].(string),
			Description: m["description"].(string),
		}
		if s, ok := m["emoji_id"].(string); ok && s != "" {
			ch.EmojiID = s
		}
		if s, ok := m["emoji_name"].(string); ok && s != "" {
			ch.EmojiName = s
		}
		out = append(out, ch)
	}
	return out
}

func flattenWelcomeChannels(v []restWelcomeChannel) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(v))
	for _, ch := range v {
		m := map[string]interface{}{
			"channel_id":  ch.ChannelID,
			"description": ch.Description,
			"emoji_id":    ch.EmojiID,
			"emoji_name":  ch.EmojiName,
		}
		out = append(out, m)
	}
	return out
}

func resourceDiscordWelcomeScreenUpsert(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	body := map[string]interface{}{
		"enabled":     d.Get("enabled").(bool),
		"description": d.Get("description").(string),
	}
	if v, ok := d.GetOk("channel"); ok {
		body["welcome_channels"] = expandWelcomeChannels(v.([]interface{}))
	} else {
		body["welcome_channels"] = []restWelcomeChannel{}
	}

	var out restWelcomeScreen
	if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s/welcome-screen", serverID), nil, body, &out); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(serverID)
	return resourceDiscordWelcomeScreenRead(ctx, d, m)
}

func resourceDiscordWelcomeScreenRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Id()
	if serverID == "" {
		serverID = d.Get("server_id").(string)
	}

	var out restWelcomeScreen
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/welcome-screen", serverID), nil, nil, &out); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId(serverID)
	_ = d.Set("server_id", serverID)
	_ = d.Set("enabled", out.Enabled)
	_ = d.Set("description", out.Description)
	_ = d.Set("channel", flattenWelcomeChannels(out.WelcomeChannels))

	return nil
}

func resourceDiscordWelcomeScreenDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Id()

	body := map[string]interface{}{
		"enabled":          false,
		"description":      "",
		"welcome_channels": []restWelcomeChannel{},
	}
	if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s/welcome-screen", serverID), nil, body, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
