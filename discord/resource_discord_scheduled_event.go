package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restScheduledEvent struct {
	ID                 string                      `json:"id"`
	GuildID            string                      `json:"guild_id"`
	ChannelID          string                      `json:"channel_id"`
	Name               string                      `json:"name"`
	Description        string                      `json:"description"`
	ScheduledStartTime string                      `json:"scheduled_start_time"`
	ScheduledEndTime   string                      `json:"scheduled_end_time"`
	PrivacyLevel       int                         `json:"privacy_level"`
	Status             int                         `json:"status"`
	EntityType         int                         `json:"entity_type"`
	EntityMetadata     *restScheduledEventMetadata `json:"entity_metadata"`
	Image              string                      `json:"image"`
}

type restScheduledEventMetadata struct {
	Location string `json:"location"`
}

func resourceDiscordScheduledEvent() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordScheduledEventCreate,
		ReadContext:   resourceDiscordScheduledEventRead,
		UpdateContext: resourceDiscordScheduledEventUpdate,
		DeleteContext: resourceDiscordScheduledEventDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"scheduled_start_time": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "RFC3339 timestamp",
			},
			"scheduled_end_time": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "RFC3339 timestamp (required for external events)",
			},
			"privacy_level": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  2,
			},
			"entity_type": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "1=stage instance, 2=voice, 3=external",
			},
			"channel_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Required for stage/voice events",
			},
			"location": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "External event location (entity_metadata.location)",
			},
			"image_data_uri": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "data: URI for event cover image",
			},
			"image_hash": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Event status (set on update to start/end/cancel where supported)",
			},
		},
	}
}

func scheduledEventPayload(d *schema.ResourceData, includeStatus bool) map[string]interface{} {
	body := map[string]interface{}{
		"name":                 d.Get("name").(string),
		"scheduled_start_time": d.Get("scheduled_start_time").(string),
		"privacy_level":        d.Get("privacy_level").(int),
		"entity_type":          d.Get("entity_type").(int),
	}
	if v := d.Get("description").(string); v != "" {
		body["description"] = v
	}
	if v := d.Get("scheduled_end_time").(string); v != "" {
		body["scheduled_end_time"] = v
	}
	if v := d.Get("channel_id").(string); v != "" {
		body["channel_id"] = v
	}
	if v := d.Get("location").(string); v != "" {
		body["entity_metadata"] = map[string]interface{}{"location": v}
	}
	if v := d.Get("image_data_uri").(string); v != "" {
		body["image"] = v
	}
	if includeStatus {
		if v, ok := d.GetOk("status"); ok {
			body["status"] = v.(int)
		}
	}
	return body
}

func resourceDiscordScheduledEventCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var out restScheduledEvent
	if err := c.DoJSON(ctx, "POST", fmt.Sprintf("/guilds/%s/scheduled-events", serverID), nil, scheduledEventPayload(d, false), &out); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(out.ID)
	return resourceDiscordScheduledEventRead(ctx, d, m)
}

func resourceDiscordScheduledEventRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var out restScheduledEvent
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/scheduled-events/%s", serverID, d.Id()), nil, nil, &out); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", out.Name)
	_ = d.Set("description", out.Description)
	_ = d.Set("scheduled_start_time", out.ScheduledStartTime)
	_ = d.Set("scheduled_end_time", out.ScheduledEndTime)
	_ = d.Set("privacy_level", out.PrivacyLevel)
	_ = d.Set("entity_type", out.EntityType)
	_ = d.Set("channel_id", out.ChannelID)
	if out.EntityMetadata != nil {
		_ = d.Set("location", out.EntityMetadata.Location)
	} else {
		_ = d.Set("location", nil)
	}
	_ = d.Set("image_hash", out.Image)
	_ = d.Set("status", out.Status)

	return nil
}

func resourceDiscordScheduledEventUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var out restScheduledEvent
	if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s/scheduled-events/%s", serverID, d.Id()), nil, scheduledEventPayload(d, true), &out); err != nil {
		return diag.FromErr(err)
	}

	return resourceDiscordScheduledEventRead(ctx, d, m)
}

func resourceDiscordScheduledEventDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	if err := c.DoJSON(ctx, "DELETE", fmt.Sprintf("/guilds/%s/scheduled-events/%s", serverID, d.Id()), nil, nil, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}
