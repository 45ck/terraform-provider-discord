package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Membership screening / member verification gate ("rules screening") configuration.
// NOTE: This endpoint is not currently covered by Discord's published OpenAPI spec,
// so this resource intentionally uses JSON passthrough and best-effort error handling.
func resourceDiscordMemberVerification() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordMemberVerificationUpsert,
		ReadContext:   resourceDiscordMemberVerificationRead,
		UpdateContext: resourceDiscordMemberVerificationUpsert,
		DeleteContext: resourceDiscordMemberVerificationDelete,
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
				Description: "JSON payload to PUT to /guilds/{guild.id}/member-verification",
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
				Description: "Normalized JSON returned by Discord",
			},
		},
	}
}

func resourceDiscordMemberVerificationUpsert(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var payload interface{}
	if err := json.Unmarshal([]byte(d.Get("payload_json").(string)), &payload); err != nil {
		return diag.FromErr(err)
	}

	if err := c.DoJSON(ctx, "PUT", fmt.Sprintf("/guilds/%s/member-verification", serverID), nil, payload, nil); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(serverID)
	return resourceDiscordMemberVerificationRead(ctx, d, m)
}

func resourceDiscordMemberVerificationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Id()
	if serverID == "" {
		serverID = d.Get("server_id").(string)
	}

	var out interface{}
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/member-verification", serverID), nil, nil, &out); err != nil {
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

func resourceDiscordMemberVerificationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Id()

	body := map[string]interface{}{
		"enabled": false,
	}
	if err := c.DoJSON(ctx, "PUT", fmt.Sprintf("/guilds/%s/member-verification", serverID), nil, body, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
