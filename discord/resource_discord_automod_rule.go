package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// AutoMod rules have multiple "union" shapes depending on trigger_type.
// This resource uses JSON passthrough to avoid an incomplete Terraform schema.
func resourceDiscordAutoModRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordAutoModRuleCreate,
		ReadContext:   resourceDiscordAutoModRuleRead,
		UpdateContext: resourceDiscordAutoModRuleUpdate,
		DeleteContext: resourceDiscordAutoModRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"payload_json": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "JSON payload to POST/PATCH AutoMod rule.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					no, oe := normalizeJSON(old)
					nn, ne := normalizeJSON(new)
					if oe != nil || ne != nil {
						return false
					}
					return no == nn
				},
			},
			"state_json": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Normalized JSON returned from the Discord API for this rule.",
			},
		},
	}
}

func resourceDiscordAutoModRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var payload interface{}
	if err := json.Unmarshal([]byte(d.Get("payload_json").(string)), &payload); err != nil {
		return diag.FromErr(err)
	}

	var out map[string]interface{}
	if err := c.DoJSON(ctx, "POST", fmt.Sprintf("/guilds/%s/auto-moderation/rules", serverID), nil, payload, &out); err != nil {
		return diag.FromErr(err)
	}

	// Expect "id" field.
	if id, ok := out["id"].(string); ok && id != "" {
		d.SetId(id)
	} else {
		return diag.Errorf("discord api did not return rule id")
	}

	return resourceDiscordAutoModRuleRead(ctx, d, m)
}

func resourceDiscordAutoModRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var out interface{}
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/auto-moderation/rules/%s", serverID, d.Id()), nil, nil, &out); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	b, err := json.Marshal(out)
	if err != nil {
		return diag.FromErr(err)
	}
	norm, err := normalizeJSON(string(b))
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("state_json", norm)
	return nil
}

func resourceDiscordAutoModRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var payload interface{}
	if err := json.Unmarshal([]byte(d.Get("payload_json").(string)), &payload); err != nil {
		return diag.FromErr(err)
	}

	if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s/auto-moderation/rules/%s", serverID, d.Id()), nil, payload, nil); err != nil {
		return diag.FromErr(err)
	}

	return resourceDiscordAutoModRuleRead(ctx, d, m)
}

func resourceDiscordAutoModRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	if err := c.DoJSON(ctx, "DELETE", fmt.Sprintf("/guilds/%s/auto-moderation/rules/%s", serverID, d.Id()), nil, nil, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
