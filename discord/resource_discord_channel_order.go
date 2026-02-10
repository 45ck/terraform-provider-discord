package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restChannelPosition struct {
	ID              string `json:"id"`
	Position        int    `json:"position,omitempty"`
	ParentID        string `json:"parent_id,omitempty"`
	LockPermissions *bool  `json:"lock_permissions,omitempty"`
}

type restGuildChannel struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	ParentID string `json:"parent_id"`
}

// discord_channel_order applies bulk channel re-ordering via PATCH /guilds/{guild.id}/channels.
// This is the reliable way to control ordering; per-channel "position" updates can be flaky.
func resourceDiscordChannelOrder() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordChannelOrderUpsert,
		ReadContext:   resourceDiscordChannelOrderRead,
		UpdateContext: resourceDiscordChannelOrderUpsert,
		DeleteContext: resourceDiscordChannelOrderDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"channel": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"channel_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"position": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"parent_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"lock_permissions": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
			"reason": {
				Type:     schema.TypeString,
				Optional: true,
				// Reason is not readable; always suppress diffs.
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool { return true },
			},
		},
	}
}

func expandChannelPositions(d *schema.ResourceData) []restChannelPosition {
	items := d.Get("channel").([]interface{})
	out := make([]restChannelPosition, 0, len(items))
	for _, it := range items {
		m := it.(map[string]interface{})
		p := restChannelPosition{
			ID:       m["channel_id"].(string),
			Position: m["position"].(int),
		}
		if v, ok := m["parent_id"].(string); ok && v != "" {
			p.ParentID = v
		}
		if v, ok := m["lock_permissions"]; ok {
			b := v.(bool)
			p.LockPermissions = &b
		}
		out = append(out, p)
	}
	return out
}

func resourceDiscordChannelOrderUpsert(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	reason := d.Get("reason").(string)

	body := expandChannelPositions(d)
	if err := c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/channels", serverID), nil, body, nil, reason); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(serverID)
	return resourceDiscordChannelOrderRead(ctx, d, m)
}

func resourceDiscordChannelOrderRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Id()
	if serverID == "" {
		serverID = d.Get("server_id").(string)
	}

	var channels []restGuildChannel
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/channels", serverID), nil, nil, &channels); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	index := map[string]restGuildChannel{}
	for _, ch := range channels {
		index[ch.ID] = ch
	}

	// Preserve config order, but refresh fields from remote for drift detection.
	items := d.Get("channel").([]interface{})
	out := make([]map[string]interface{}, 0, len(items))
	for _, it := range items {
		mm := it.(map[string]interface{})
		id := mm["channel_id"].(string)
		ch, ok := index[id]
		if !ok {
			return diag.Errorf("channel_id %s not found in server %s", id, serverID)
		}
		out = append(out, map[string]interface{}{
			"channel_id":       id,
			"position":         ch.Position,
			"parent_id":        ch.ParentID,
			"lock_permissions": mm["lock_permissions"],
		})
	}

	d.SetId(serverID)
	_ = d.Set("server_id", serverID)
	_ = d.Set("channel", out)
	return nil
}

func resourceDiscordChannelOrderDelete(ctx context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	// No-op. Ordering is not meaningfully "deletable".
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "discord_channel_order does not revert ordering on destroy",
	}}
}
