package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDiscordSoundboardDefaultSounds() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDiscordSoundboardDefaultSoundsRead,
		Schema: map[string]*schema.Schema{
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

func dataSourceDiscordSoundboardDefaultSoundsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	var out []restSoundboardSound
	if err := c.DoJSON(ctx, "GET", "/soundboard-default-sounds", nil, nil, &out); err != nil {
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

	d.SetId(fmt.Sprintf("%d", len(sounds)))
	_ = d.Set("sound", sounds)
	return nil
}
