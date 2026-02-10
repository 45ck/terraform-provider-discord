package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restEmojiLite struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Managed  bool   `json:"managed"`
	Animated bool   `json:"animated"`
}

func dataSourceDiscordEmojis() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDiscordEmojisRead,
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"emoji": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":       {Type: schema.TypeString, Computed: true},
						"name":     {Type: schema.TypeString, Computed: true},
						"managed":  {Type: schema.TypeBool, Computed: true},
						"animated": {Type: schema.TypeBool, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceDiscordEmojisRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var out []restEmojiLite
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/emojis", serverID), nil, nil, &out); err != nil {
		return diag.FromErr(err)
	}

	emojis := make([]map[string]interface{}, 0, len(out))
	for _, e := range out {
		emojis = append(emojis, map[string]interface{}{
			"id":       e.ID,
			"name":     e.Name,
			"managed":  e.Managed,
			"animated": e.Animated,
		})
	}

	d.SetId(serverID)
	_ = d.Set("emoji", emojis)
	return nil
}
