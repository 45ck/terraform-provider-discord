package discord

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restGuildDS struct {
	ID                          string `json:"id"`
	Name                        string `json:"name"`
	Region                      string `json:"region"`
	DefaultMessageNotifications int    `json:"default_message_notifications"`
	VerificationLevel           int    `json:"verification_level"`
	ExplicitContentFilter       int    `json:"explicit_content_filter"`
	AfkTimeout                  int    `json:"afk_timeout"`
	AfkChannelID                string `json:"afk_channel_id"`
	OwnerID                     string `json:"owner_id"`
	Icon                        string `json:"icon"`
	Splash                      string `json:"splash"`
	SystemChannelID             string `json:"system_channel_id"`
}

func dataSourceDiscordServer() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDiscordServerRead,
		Schema: map[string]*schema.Schema{
			"server_id": {
				ExactlyOneOf: []string{"server_id", "name"},
				Type:         schema.TypeString,
				Optional:     true,
			},
			"name": {
				ExactlyOneOf: []string{"server_id", "name"},
				Type:         schema.TypeString,
				Optional:     true,
			},
			"region": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"default_message_notifications": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"verification_level": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"explicit_content_filter": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"afk_timeout": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"icon_hash": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"splash_hash": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"afk_channel_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceDiscordServerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	if _, ok := d.GetOk("name"); ok {
		// Provider tokens are bot tokens (Authorization: Bot ...). Discord does not provide a bot-safe API
		// to enumerate guilds by name.
		return diag.Errorf("discord_server data source does not support lookup by name for bot tokens; set server_id")
	}

	serverID := d.Get("server_id").(string)
	if serverID == "" {
		return diag.Errorf("either server_id or name must be set")
	}

	var guild restGuildDS
	if err := c.DoJSON(ctx, "GET", "/guilds/"+serverID, nil, nil, &guild); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(guild.ID)
	_ = d.Set("server_id", guild.ID)
	_ = d.Set("name", guild.Name)
	_ = d.Set("region", guild.Region)
	_ = d.Set("afk_timeout", guild.AfkTimeout)
	_ = d.Set("icon_hash", guild.Icon)
	_ = d.Set("splash_hash", guild.Splash)
	_ = d.Set("default_message_notifications", guild.DefaultMessageNotifications)
	_ = d.Set("verification_level", guild.VerificationLevel)
	_ = d.Set("explicit_content_filter", guild.ExplicitContentFilter)
	if guild.AfkChannelID != "" {
		_ = d.Set("afk_channel_id", guild.AfkChannelID)
	}
	if guild.OwnerID != "" {
		_ = d.Set("owner_id", guild.OwnerID)
	}
	return nil
}
