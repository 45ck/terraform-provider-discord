package discord

import (
	"context"
	"github.com/andersfylling/disgord"
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
	var diags diag.Diagnostics
	var err error
	var server *disgord.Guild
	client := m.(*Context).Client

	serverId := d.Get("server_id").(string)
	server, err = client.GetGuild(ctx, getId(serverId))
	if err != nil {
		return diag.Errorf("Failed to fetch server %s: %s", serverId, err.Error())
	}

	d.SetId(server.ID.String())
	if server.SystemChannelID.IsZero() {
		d.Set("system_channel_id", "")
	} else {
		d.Set("system_channel_id", server.SystemChannelID.String())
	}

	return diags
}
