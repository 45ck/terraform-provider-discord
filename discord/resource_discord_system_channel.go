package discord

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDiscordSystemChannel() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSystemChannelUpsert,
		ReadContext:   resourceSystemChannelRead,
		UpdateContext: resourceSystemChannelUpsert,
		DeleteContext: resourceSystemChannelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"system_channel_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"reason": {
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool { return true },
			},
		},
	}
}

func resourceSystemChannelUpsert(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	systemID := d.Get("system_channel_id").(string)
	reason := d.Get("reason").(string)

	body := map[string]interface{}{
		"system_channel_id": systemID,
	}
	if err := c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s", serverID), nil, body, nil, reason); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(serverID)
	return resourceSystemChannelRead(ctx, d, m)
}

func resourceSystemChannelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Id()
	if serverID == "" {
		serverID = d.Get("server_id").(string)
	}

	var guild restGuildDS
	if err := c.DoJSON(ctx, "GET", "/guilds/"+serverID, nil, nil, &guild); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId(guild.ID)
	_ = d.Set("server_id", guild.ID)
	_ = d.Set("system_channel_id", guild.SystemChannelID)
	return nil
}

func resourceSystemChannelDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	reason := d.Get("reason").(string)

	body := map[string]interface{}{
		"system_channel_id": nil,
	}
	if err := c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s", serverID), nil, body, nil, reason); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
