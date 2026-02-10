package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restWebhook struct {
	ID        string `json:"id"`
	Type      int    `json:"type"`
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
	Name      string `json:"name"`
	Token     string `json:"token"`
	URL       string `json:"url"`
}

func resourceDiscordWebhook() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordWebhookCreate,
		ReadContext:   resourceDiscordWebhookRead,
		UpdateContext: resourceDiscordWebhookUpdate,
		DeleteContext: resourceDiscordWebhookDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"channel_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"avatar_data_uri": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "data: URI for the webhook avatar",
			},
			"token": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"guild_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceDiscordWebhookCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	channelID := d.Get("channel_id").(string)

	body := map[string]interface{}{
		"name": d.Get("name").(string),
	}
	if v := d.Get("avatar_data_uri").(string); v != "" {
		body["avatar"] = v
	}

	var out restWebhook
	if err := c.DoJSON(ctx, "POST", fmt.Sprintf("/channels/%s/webhooks", channelID), nil, body, &out); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(out.ID)
	return resourceDiscordWebhookRead(ctx, d, m)
}

func resourceDiscordWebhookRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	var out restWebhook
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/webhooks/%s", d.Id()), nil, nil, &out); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("channel_id", out.ChannelID)
	_ = d.Set("guild_id", out.GuildID)
	_ = d.Set("name", out.Name)
	_ = d.Set("token", out.Token)
	_ = d.Set("url", out.URL)

	return nil
}

func resourceDiscordWebhookUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	body := map[string]interface{}{
		"name": d.Get("name").(string),
	}
	if d.HasChange("channel_id") {
		body["channel_id"] = d.Get("channel_id").(string)
	}
	if d.HasChange("avatar_data_uri") {
		v := d.Get("avatar_data_uri").(string)
		if v == "" {
			body["avatar"] = nil
		} else {
			body["avatar"] = v
		}
	}

	var out restWebhook
	if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/webhooks/%s", d.Id()), nil, body, &out); err != nil {
		return diag.FromErr(err)
	}
	return resourceDiscordWebhookRead(ctx, d, m)
}

func resourceDiscordWebhookDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	if err := c.DoJSON(ctx, "DELETE", fmt.Sprintf("/webhooks/%s", d.Id()), nil, nil, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
