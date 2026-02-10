package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restStickerLite struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	FormatType  int    `json:"format_type"`
}

func dataSourceDiscordStickers() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDiscordStickersRead,
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"sticker": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":          {Type: schema.TypeString, Computed: true},
						"name":        {Type: schema.TypeString, Computed: true},
						"description": {Type: schema.TypeString, Computed: true},
						"tags":        {Type: schema.TypeString, Computed: true},
						"format_type": {Type: schema.TypeInt, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceDiscordStickersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var out []restStickerLite
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/stickers", serverID), nil, nil, &out); err != nil {
		return diag.FromErr(err)
	}

	stickers := make([]map[string]interface{}, 0, len(out))
	for _, s := range out {
		stickers = append(stickers, map[string]interface{}{
			"id":          s.ID,
			"name":        s.Name,
			"description": s.Description,
			"tags":        s.Tags,
			"format_type": s.FormatType,
		})
	}

	d.SetId(serverID)
	_ = d.Set("sticker", stickers)
	return nil
}
