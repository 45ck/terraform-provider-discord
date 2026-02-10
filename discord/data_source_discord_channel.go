package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restChannelLite struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     uint   `json:"type"`
	ParentID string `json:"parent_id"`
}

func dataSourceDiscordChannel() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDiscordChannelRead,
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional channel type filter (text, voice, category, news, stage, forum, media).",
			},
			"parent_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceDiscordChannelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	name := d.Get("name").(string)
	wantType := d.Get("type").(string)

	var channels []restChannelLite
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/channels", serverID), nil, nil, &channels); err != nil {
		return diag.FromErr(err)
	}

	var matches []restChannelLite
	for _, ch := range channels {
		if ch.Name != name {
			continue
		}
		if wantType != "" {
			got, ok := getTextChannelType(ch.Type)
			if !ok || got != wantType {
				continue
			}
		}
		matches = append(matches, ch)
	}

	if len(matches) == 0 {
		return diag.Errorf("no channel named %q found in server %s", name, serverID)
	}
	if len(matches) > 1 {
		return diag.Errorf("multiple channels named %q found in server %s; specify a more precise filter", name, serverID)
	}

	ch := matches[0]
	d.SetId(ch.ID)
	_ = d.Set("parent_id", ch.ParentID)
	return nil
}
