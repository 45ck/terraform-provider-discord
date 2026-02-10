package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDiscordSoundboardSounds() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDiscordSoundboardSoundsRead,
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"sound": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sound_id":   {Type: schema.TypeString, Computed: true},
						"name":       {Type: schema.TypeString, Computed: true},
						"volume":     {Type: schema.TypeFloat, Computed: true},
						"emoji_id":   {Type: schema.TypeString, Computed: true},
						"emoji_name": {Type: schema.TypeString, Computed: true},
						"available":  {Type: schema.TypeBool, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceDiscordSoundboardSoundsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var out []restSoundboardSound
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/soundboard-sounds", serverID), nil, nil, &out); err != nil {
		return diag.FromErr(err)
	}

	sounds := make([]map[string]interface{}, 0, len(out))
	for _, s := range out {
		sounds = append(sounds, map[string]interface{}{
			"sound_id":   s.SoundID,
			"name":       s.Name,
			"volume":     s.Volume,
			"emoji_id":   s.EmojiID,
			"emoji_name": s.EmojiName,
			"available":  s.Available,
		})
	}

	d.SetId(serverID)
	_ = d.Set("sound", sounds)
	return nil
}
