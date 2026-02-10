package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restStageInstance struct {
	ID                    string `json:"id"`
	ChannelID             string `json:"channel_id"`
	GuildID               string `json:"guild_id"`
	Topic                 string `json:"topic"`
	PrivacyLevel          int    `json:"privacy_level"`
	GuildScheduledEventID string `json:"guild_scheduled_event_id"`
}

func resourceDiscordStageInstance() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordStageInstanceCreate,
		ReadContext:   resourceDiscordStageInstanceRead,
		UpdateContext: resourceDiscordStageInstanceUpdate,
		DeleteContext: resourceDiscordStageInstanceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"channel_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"topic": {
				Type:     schema.TypeString,
				Required: true,
			},
			"privacy_level": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  2,
			},
			"send_start_notification": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"scheduled_event_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"server_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceDiscordStageInstanceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	body := map[string]interface{}{
		"channel_id":    d.Get("channel_id").(string),
		"topic":         d.Get("topic").(string),
		"privacy_level": d.Get("privacy_level").(int),
	}
	if v, ok := d.GetOk("send_start_notification"); ok {
		body["send_start_notification"] = v.(bool)
	}
	if v, ok := d.GetOk("scheduled_event_id"); ok && v.(string) != "" {
		body["guild_scheduled_event_id"] = v.(string)
	}

	var out restStageInstance
	if err := c.DoJSON(ctx, "POST", "/stage-instances", nil, body, &out); err != nil {
		return diag.FromErr(err)
	}

	// Resource is keyed by channel_id in the API.
	d.SetId(d.Get("channel_id").(string))
	return resourceDiscordStageInstanceRead(ctx, d, m)
}

func resourceDiscordStageInstanceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	var out restStageInstance
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/stage-instances/%s", d.Id()), nil, nil, &out); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("channel_id", out.ChannelID)
	_ = d.Set("topic", out.Topic)
	_ = d.Set("privacy_level", out.PrivacyLevel)
	_ = d.Set("scheduled_event_id", out.GuildScheduledEventID)
	_ = d.Set("server_id", out.GuildID)
	return nil
}

func resourceDiscordStageInstanceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	body := map[string]interface{}{}
	if d.HasChange("topic") {
		body["topic"] = d.Get("topic").(string)
	}
	if d.HasChange("privacy_level") {
		body["privacy_level"] = d.Get("privacy_level").(int)
	}
	if len(body) == 0 {
		return nil
	}

	if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/stage-instances/%s", d.Id()), nil, body, nil); err != nil {
		return diag.FromErr(err)
	}
	return resourceDiscordStageInstanceRead(ctx, d, m)
}

func resourceDiscordStageInstanceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	if err := c.DoJSON(ctx, "DELETE", fmt.Sprintf("/stage-instances/%s", d.Id()), nil, nil, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
