package discord

import (
	"github.com/andersfylling/disgord"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/context"
)

func resourceDiscordSystemChannel() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSystemChannelCreate,
		ReadContext:   resourceSystemChannelRead,
		UpdateContext: resourceSystemChannelUpdate,
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
		},
	}
}

func resourceSystemChannelCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	serverId := disgord.NewSnowflake(0)
	if v, ok := d.GetOk("server_id"); ok {
		serverId = disgord.ParseSnowflakeString(v.(string))
	}

	server, err := client.GetGuild(ctx, serverId)
	if err != nil {
		return diag.Errorf("Failed to find server: %s", err.Error())
	}

	builder := client.UpdateGuild(ctx, server.ID)

	if v, ok := d.GetOk("system_channel_id"); ok {
		builder.SetSystemChannelID(disgord.ParseSnowflakeString(v.(string)))
	} else {
		return diag.Errorf("system_channel_id is required")
	}

	server, err = builder.Execute()
	if err != nil {
		return diag.Errorf("Failed to edit server: %s", err.Error())
	}

	d.SetId(d.Get("server_id").(string))

	return diags
}

func resourceSystemChannelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	serverId := disgord.ParseSnowflakeString(d.Id())

	server, err := client.GetGuild(ctx, serverId)
	if err != nil {
		return diag.Errorf("Error fetching server: %s", err.Error())
	}

	d.Set("system_channel_id", server.SystemChannelID.String())

	return diags
}

func resourceSystemChannelUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	serverId := disgord.ParseSnowflakeString(d.Get("server_id").(string))

	server, err := client.GetGuild(ctx, serverId)
	if err != nil {
		return diag.Errorf("Error fetching server: %s", err.Error())
	}

	builder := client.UpdateGuild(ctx, server.ID)
	edit := false

	if d.HasChange("system_channel_id") {
		id := d.Get("system_channel_id").(string)
		builder.SetSystemChannelID(disgord.ParseSnowflakeString(id))
		edit = true
	}

	if edit {
		_, err = builder.Execute()
		if err != nil {
			return diag.Errorf("Failed to edit server: %s", err.Error())
		}
	}

	return diags
}

func resourceSystemChannelDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	serverId := disgord.ParseSnowflakeString(d.Get("server_id").(string))

	server, err := client.GetGuild(ctx, serverId)
	if err != nil {
		return diag.Errorf("Error fetching server: %s", err.Error())
	}

	builder := client.UpdateGuild(ctx, server.ID)

	// Clear system channel (treat delete as reverting to "no system channel").
	builder.SetSystemChannelID(disgord.NewSnowflake(0))

	_, err = builder.Execute()
	if err != nil {
		return diag.Errorf("Failed to edit server: %s", err.Error())
	}

	return diags
}
