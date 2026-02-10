package discord

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDiscordSystemChannel() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDiscordSystemChannelRead,
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"system_channel_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceDiscordSystemChannelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var guild restGuildDS
	if err := c.DoJSON(ctx, "GET", "/guilds/"+serverID, nil, nil, &guild); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(guild.ID)
	if guild.SystemChannelID == "" {
		_ = d.Set("system_channel_id", "")
	} else {
		_ = d.Set("system_channel_id", guild.SystemChannelID)
	}
	return nil
}
