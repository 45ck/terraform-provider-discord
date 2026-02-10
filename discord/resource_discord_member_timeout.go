package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restGuildMember struct {
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	CommunicationDisabledUntil string `json:"communication_disabled_until"`
	Nick                       string `json:"nick"`
}

func resourceDiscordMemberTimeout() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordMemberTimeoutUpsert,
		ReadContext:   resourceDiscordMemberTimeoutRead,
		UpdateContext: resourceDiscordMemberTimeoutUpsert,
		DeleteContext: resourceDiscordMemberTimeoutDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"user_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"until": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "RFC3339 timestamp for communication_disabled_until. Use an empty string to clear.",
			},
			"reason": {
				Type:     schema.TypeString,
				Optional: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return true
				},
			},
		},
	}
}

func resourceDiscordMemberTimeoutUpsert(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	userID := d.Get("user_id").(string)

	until := d.Get("until").(string)
	var val interface{}
	if until == "" {
		val = nil
	} else {
		val = until
	}

	body := map[string]interface{}{
		"communication_disabled_until": val,
	}

	reason := d.Get("reason").(string)
	if err := c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/members/%s", serverID, userID), nil, body, nil, reason); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s:%s", serverID, userID))
	return resourceDiscordMemberTimeoutRead(ctx, d, m)
}

func resourceDiscordMemberTimeoutRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID, userID, err := parseTwoIds(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var out restGuildMember
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/members/%s", serverID, userID), nil, nil, &out); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("server_id", serverID)
	_ = d.Set("user_id", userID)
	_ = d.Set("until", out.CommunicationDisabledUntil)
	return nil
}

func resourceDiscordMemberTimeoutDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Clear timeout.
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	userID := d.Get("user_id").(string)
	reason := d.Get("reason").(string)

	body := map[string]interface{}{
		"communication_disabled_until": nil,
	}
	if err := c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/members/%s", serverID, userID), nil, body, nil, reason); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
