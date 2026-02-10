package discord

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/polds/imgbase64"
)

type restGuildCreate struct {
	Name                        string `json:"name"`
	Region                      string `json:"region,omitempty"`
	Icon                        string `json:"icon,omitempty"`
	VerificationLevel           int    `json:"verification_level,omitempty"`
	DefaultMessageNotifications int    `json:"default_message_notifications,omitempty"`
	ExplicitContentFilter       int    `json:"explicit_content_filter,omitempty"`
	Channels                    []any  `json:"channels,omitempty"`
}

type restGuildUpdate struct {
	Name                        *string `json:"name,omitempty"`
	Region                      *string `json:"region,omitempty"`
	Icon                        *string `json:"icon,omitempty"`
	Splash                      *string `json:"splash,omitempty"`
	VerificationLevel           *int    `json:"verification_level,omitempty"`
	DefaultMessageNotifications *int    `json:"default_message_notifications,omitempty"`
	ExplicitContentFilter       *int    `json:"explicit_content_filter,omitempty"`
	AfkChannelID                *string `json:"afk_channel_id,omitempty"`
	AfkTimeout                  *int    `json:"afk_timeout,omitempty"`
	OwnerID                     *string `json:"owner_id,omitempty"`
}

type restGuildChannelID struct {
	ID string `json:"id"`
}

func resourceDiscordServer() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServerCreate,
		ReadContext:   resourceServerRead,
		UpdateContext: resourceServerUpdate,
		DeleteContext: resourceServerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"region": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"verification_level": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
				ValidateFunc: func(val interface{}, key string) (warns []string, errors []error) {
					v := val.(int)
					if v > 3 || v < 0 {
						errors = append(errors, fmt.Errorf("verification_level must be between 0 and 3 inclusive, got: %d", v))
					}
					return
				},
			},
			"explicit_content_filter": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
				ValidateFunc: func(val interface{}, key string) (warns []string, errors []error) {
					v := val.(int)
					if v > 2 || v < 0 {
						errors = append(errors, fmt.Errorf("explicit_content_filter must be between 0 and 2 inclusive, got: %d", v))
					}
					return
				},
			},
			"default_message_notifications": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
				ValidateFunc: func(val interface{}, key string) (warns []string, errors []error) {
					v := val.(int)
					if v != 0 && v != 1 {
						errors = append(errors, fmt.Errorf("default_message_notifications must be 0 or 1, got: %d", v))
					}
					return
				},
			},
			"afk_channel_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"afk_timeout": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  300,
				ValidateFunc: func(val interface{}, key string) (warns []string, errors []error) {
					v := val.(int)
					if v < 0 {
						errors = append(errors, fmt.Errorf("afk_timeout must be greater than 0, got: %d", v))
					}
					return
				},
			},
			"icon_url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"icon_data_uri": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"icon_hash": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"splash_url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"splash_data_uri": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"splash_hash": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceServerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	icon := ""
	if v, ok := d.GetOk("icon_url"); ok {
		icon = imgbase64.FromRemote(v.(string))
	}
	if v, ok := d.GetOk("icon_data_uri"); ok {
		icon = v.(string)
	}

	name := d.Get("name").(string)
	req := restGuildCreate{
		Name:                        name,
		Region:                      d.Get("region").(string),
		Icon:                        icon,
		VerificationLevel:           d.Get("verification_level").(int),
		DefaultMessageNotifications: d.Get("default_message_notifications").(int),
		ExplicitContentFilter:       d.Get("explicit_content_filter").(int),
	}

	var guild restGuildDS
	if err := c.DoJSON(ctx, "POST", "/guilds", nil, req, &guild); err != nil {
		return diag.FromErr(err)
	}

	// Remove default channels that Discord creates on guild creation.
	var channels []restGuildChannelID
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/channels", guild.ID), nil, nil, &channels); err == nil {
		for _, ch := range channels {
			_ = c.DoJSON(ctx, "DELETE", fmt.Sprintf("/channels/%s", ch.ID), nil, nil, nil)
		}
	}

	// Apply optional post-create settings (splash, afk, owner).
	splash := ""
	if v, ok := d.GetOk("splash_url"); ok {
		splash = imgbase64.FromRemote(v.(string))
	}
	if v, ok := d.GetOk("splash_data_uri"); ok {
		splash = v.(string)
	}

	up := restGuildUpdate{}
	edit := false
	if v, ok := d.GetOk("afk_channel_id"); ok {
		s := v.(string)
		up.AfkChannelID = &s
		edit = true
	}
	if v, ok := d.GetOk("afk_timeout"); ok {
		i := v.(int)
		up.AfkTimeout = &i
		edit = true
	}
	if v, ok := d.GetOk("owner_id"); ok {
		s := v.(string)
		up.OwnerID = &s
		edit = true
	}
	if splash != "" {
		up.Splash = &splash
		edit = true
	}
	if edit {
		if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s", guild.ID), nil, up, &guild); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(guild.ID)
	_ = d.Set("server_id", guild.ID)
	_ = d.Set("icon_hash", guild.Icon)
	_ = d.Set("splash_hash", guild.Splash)
	return resourceServerRead(ctx, d, m)
}

func resourceServerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	var guild restGuildDS
	if err := c.DoJSON(ctx, "GET", "/guilds/"+d.Id(), nil, nil, &guild); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("server_id", guild.ID)
	_ = d.Set("name", guild.Name)
	_ = d.Set("region", guild.Region)
	_ = d.Set("default_message_notifications", guild.DefaultMessageNotifications)
	_ = d.Set("afk_timeout", guild.AfkTimeout)
	_ = d.Set("icon_hash", guild.Icon)
	_ = d.Set("splash_hash", guild.Splash)
	_ = d.Set("verification_level", guild.VerificationLevel)
	_ = d.Set("explicit_content_filter", guild.ExplicitContentFilter)
	if guild.AfkChannelID != "" {
		_ = d.Set("afk_channel_id", guild.AfkChannelID)
	}

	// Do not backfill owner_id unless configured to avoid forcing ownership changes.
	if d.Get("owner_id").(string) != "" && guild.OwnerID != "" {
		_ = d.Set("owner_id", guild.OwnerID)
	}

	return nil
}

func resourceServerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	up := restGuildUpdate{}
	edit := false

	if d.HasChange("icon_url") {
		s := imgbase64.FromRemote(d.Get("icon_url").(string))
		up.Icon = &s
		edit = true
	}
	if d.HasChange("icon_data_uri") {
		s := d.Get("icon_data_uri").(string)
		up.Icon = &s
		edit = true
	}
	if d.HasChange("splash_url") {
		s := imgbase64.FromRemote(d.Get("splash_url").(string))
		up.Splash = &s
		edit = true
	}
	if d.HasChange("splash_data_uri") {
		s := d.Get("splash_data_uri").(string)
		up.Splash = &s
		edit = true
	}
	if d.HasChange("afk_channel_id") {
		s := d.Get("afk_channel_id").(string)
		up.AfkChannelID = &s
		edit = true
	}
	if d.HasChange("afk_timeout") {
		i := d.Get("afk_timeout").(int)
		up.AfkTimeout = &i
		edit = true
	}
	if d.HasChange("owner_id") {
		s := d.Get("owner_id").(string)
		up.OwnerID = &s
		edit = true
	}
	if d.HasChange("verification_level") {
		i := d.Get("verification_level").(int)
		up.VerificationLevel = &i
		edit = true
	}
	if d.HasChange("default_message_notifications") {
		i := d.Get("default_message_notifications").(int)
		up.DefaultMessageNotifications = &i
		edit = true
	}
	if d.HasChange("explicit_content_filter") {
		i := d.Get("explicit_content_filter").(int)
		up.ExplicitContentFilter = &i
		edit = true
	}
	if d.HasChange("name") {
		s := d.Get("name").(string)
		up.Name = &s
		edit = true
	}
	if d.HasChange("region") {
		s := d.Get("region").(string)
		up.Region = &s
		edit = true
	}

	if edit {
		var guild restGuildDS
		if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s", d.Id()), nil, up, &guild); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceServerRead(ctx, d, m)
}

func resourceServerDelete(ctx context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	// No-op by default; deleting a guild is typically undesirable/dangerous and may not be permitted for bot tokens.
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "discord_server does not delete the guild on destroy",
	}}
}
