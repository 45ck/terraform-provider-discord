package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restRolePosition struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

type restRole struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

// discord_role_order applies bulk role ordering via PATCH /guilds/{guild.id}/roles.
// Role order is critical for permissions. This resource should typically be used with prevent_destroy.
func resourceDiscordRoleOrder() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordRoleOrderUpsert,
		ReadContext:   resourceDiscordRoleOrderRead,
		UpdateContext: resourceDiscordRoleOrderUpsert,
		DeleteContext: resourceDiscordRoleOrderDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"role": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"position": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},
			"reason": {
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool { return true },
			},
		},
	}
}

func expandRolePositions(d *schema.ResourceData) []restRolePosition {
	items := d.Get("role").([]interface{})
	out := make([]restRolePosition, 0, len(items))
	for _, it := range items {
		m := it.(map[string]interface{})
		out = append(out, restRolePosition{
			ID:       m["role_id"].(string),
			Position: m["position"].(int),
		})
	}
	return out
}

func resourceDiscordRoleOrderUpsert(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	reason := d.Get("reason").(string)

	body := expandRolePositions(d)
	var out []restRole
	if err := c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/roles", serverID), nil, body, &out, reason); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(serverID)
	return resourceDiscordRoleOrderRead(ctx, d, m)
}

func resourceDiscordRoleOrderRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	serverID := d.Id()
	if serverID == "" {
		serverID = d.Get("server_id").(string)
	}

	// The most reliable read is to GET the guild and inspect roles, but that is an extra schema.
	// Use the existing disgord client for roles here for stability.
	client := m.(*Context).Client
	guild, err := client.GetGuild(ctx, getId(serverID))
	if err != nil {
		return diag.FromErr(err)
	}

	index := map[string]int{}
	for _, r := range guild.Roles {
		index[r.ID.String()] = r.Position
	}

	items := d.Get("role").([]interface{})
	out := make([]map[string]interface{}, 0, len(items))
	for _, it := range items {
		mm := it.(map[string]interface{})
		id := mm["role_id"].(string)
		pos, ok := index[id]
		if !ok {
			return diag.Errorf("role_id %s not found in server %s", id, serverID)
		}
		out = append(out, map[string]interface{}{
			"role_id":  id,
			"position": pos,
		})
	}

	d.SetId(serverID)
	_ = d.Set("server_id", serverID)
	_ = d.Set("role", out)
	return nil
}

func resourceDiscordRoleOrderDelete(ctx context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "discord_role_order does not revert ordering on destroy",
	}}
}
