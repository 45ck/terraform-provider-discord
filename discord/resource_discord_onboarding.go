package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// This resource intentionally uses a JSON payload passthrough.
// Onboarding's schema is fairly deep and changes over time; keeping this as raw JSON
// avoids pinning users to an incomplete/incorrect Terraform schema.
func resourceDiscordOnboarding() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordOnboardingUpsert,
		ReadContext:   resourceDiscordOnboardingRead,
		UpdateContext: resourceDiscordOnboardingUpsert,
		DeleteContext: resourceDiscordOnboardingDelete,
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
				Description: "JSON payload to PATCH to /guilds/{guild.id}/onboarding",
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
				Description: "Normalized JSON returned from GET /guilds/{guild.id}/onboarding",
			},
		},
	}
}

func resourceDiscordOnboardingUpsert(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var payload interface{}
	if err := json.Unmarshal([]byte(d.Get("payload_json").(string)), &payload); err != nil {
		return diag.FromErr(err)
	}

	if err := c.DoJSON(ctx, "PUT", fmt.Sprintf("/guilds/%s/onboarding", serverID), nil, payload, nil); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(serverID)
	return resourceDiscordOnboardingRead(ctx, d, m)
}

func resourceDiscordOnboardingRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Id()
	if serverID == "" {
		serverID = d.Get("server_id").(string)
	}

	var out interface{}
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/onboarding", serverID), nil, nil, &out); err != nil {
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

func resourceDiscordOnboardingDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Id()

	// Best-effort disable. Users that want to "remove" onboarding should explicitly manage enabled=false.
	body := map[string]interface{}{"enabled": false}
	if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s/onboarding", serverID), nil, body, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
