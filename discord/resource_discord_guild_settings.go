package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// discord_guild_settings is a "full coverage" escape hatch for guild-level settings that are not
// represented by the legacy discord_server resource schema.
//
// It PATCHes /guilds/{guild_id} with payload_json and stores the resulting GET /guilds/{guild_id}
// response as state_json (normalized).
func resourceDiscordGuildSettings() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordGuildSettingsUpsert,
		ReadContext:   resourceDiscordGuildSettingsRead,
		UpdateContext: resourceDiscordGuildSettingsUpsert,
		DeleteContext: resourceDiscordGuildSettingsDelete,
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
				Description: "JSON payload to PATCH to /guilds/{guild.id}",
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
				Description: "Normalized JSON returned from GET /guilds/{guild.id}",
			},
		},
	}
}

func resourceDiscordGuildSettingsUpsert(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var payload interface{}
	if err := json.Unmarshal([]byte(d.Get("payload_json").(string)), &payload); err != nil {
		return diag.FromErr(err)
	}

	// Discord returns the updated guild object.
	var out interface{}
	if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s", serverID), nil, payload, &out); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(serverID)
	return resourceDiscordGuildSettingsRead(ctx, d, m)
}

func resourceDiscordGuildSettingsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Id()
	if serverID == "" {
		serverID = d.Get("server_id").(string)
	}

	var out interface{}
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s", serverID), nil, nil, &out); err != nil {
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

	d.SetId(serverID)
	_ = d.Set("server_id", serverID)
	_ = d.Set("state_json", norm)
	return nil
}

func resourceDiscordGuildSettingsDelete(ctx context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	// Intentionally no-op: Terraform "destroy" should not attempt to revert all guild settings
	// (there is no safe baseline). Users can manage explicit reversions via payload_json changes.
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "discord_guild_settings does not revert guild settings on destroy",
	}}
}
