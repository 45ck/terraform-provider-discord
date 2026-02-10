package discord

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"os"
)

type restSoundboardSound struct {
	Name      string  `json:"name"`
	SoundID   string  `json:"sound_id"`
	Volume    float64 `json:"volume"`
	EmojiID   string  `json:"emoji_id"`
	EmojiName string  `json:"emoji_name"`
	GuildID   string  `json:"guild_id"`
	Available bool    `json:"available"`
}

func resourceDiscordSoundboardSound() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordSoundboardSoundCreate,
		ReadContext:   resourceDiscordSoundboardSoundRead,
		UpdateContext: resourceDiscordSoundboardSoundUpdate,
		DeleteContext: resourceDiscordSoundboardSoundDelete,
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
			"volume": {
				Type:     schema.TypeFloat,
				Optional: true,
				Default:  1.0,
			},
			"emoji_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"emoji_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"sound_file_path": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Path to the sound file. Sent as base64 in the create request.",
			},
			"available": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"reason": {
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool { return true },
			},
		},
	}
}

func resourceDiscordSoundboardSoundCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	reason := d.Get("reason").(string)

	b, err := os.ReadFile(d.Get("sound_file_path").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	snd := base64.StdEncoding.EncodeToString(b)

	body := map[string]interface{}{
		"name":   d.Get("name").(string),
		"sound":  snd,
		"volume": d.Get("volume").(float64),
	}
	if v := d.Get("emoji_id").(string); v != "" {
		body["emoji_id"] = v
	}
	if v := d.Get("emoji_name").(string); v != "" {
		body["emoji_name"] = v
	}

	var out restSoundboardSound
	if err := c.DoJSONWithReason(ctx, "POST", fmt.Sprintf("/guilds/%s/soundboard-sounds", serverID), nil, body, &out, reason); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(out.SoundID)
	return resourceDiscordSoundboardSoundRead(ctx, d, m)
}

func resourceDiscordSoundboardSoundRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var out restSoundboardSound
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/soundboard-sounds/%s", serverID, d.Id()), nil, nil, &out); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", out.Name)
	_ = d.Set("volume", out.Volume)
	_ = d.Set("emoji_id", out.EmojiID)
	_ = d.Set("emoji_name", out.EmojiName)
	_ = d.Set("available", out.Available)
	return nil
}

func resourceDiscordSoundboardSoundUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	reason := d.Get("reason").(string)

	body := map[string]interface{}{}
	if d.HasChange("name") {
		body["name"] = d.Get("name").(string)
	}
	if d.HasChange("volume") {
		body["volume"] = d.Get("volume").(float64)
	}
	if d.HasChange("emoji_id") {
		v := d.Get("emoji_id").(string)
		if v == "" {
			body["emoji_id"] = nil
		} else {
			body["emoji_id"] = v
		}
	}
	if d.HasChange("emoji_name") {
		v := d.Get("emoji_name").(string)
		if v == "" {
			body["emoji_name"] = nil
		} else {
			body["emoji_name"] = v
		}
	}
	if len(body) == 0 {
		return nil
	}

	var out restSoundboardSound
	if err := c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/soundboard-sounds/%s", serverID, d.Id()), nil, body, &out, reason); err != nil {
		return diag.FromErr(err)
	}

	return resourceDiscordSoundboardSoundRead(ctx, d, m)
}

func resourceDiscordSoundboardSoundDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	reason := d.Get("reason").(string)

	if err := c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/guilds/%s/soundboard-sounds/%s", serverID, d.Id()), nil, nil, nil, reason); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
