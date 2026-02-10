package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/url"
)

// discord_ban manages a guild ban.
// This is powerful and potentially disruptive; use it intentionally.
func resourceDiscordBan() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordBanCreate,
		ReadContext:   resourceDiscordBanRead,
		DeleteContext: resourceDiscordBanDelete,
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
			// create-only knob. Discord does not persist this setting for reads.
			"delete_message_seconds": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "How many seconds of messages to delete (0 to not delete).",
			},
			// audit log reason: not readable; ignore diffs.
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

func banID(serverID, userID string) string {
	return fmt.Sprintf("%s:%s", serverID, userID)
}

func resourceDiscordBanCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	userID := d.Get("user_id").(string)

	q := url.Values{}
	if v, ok := d.GetOk("delete_message_seconds"); ok {
		q.Set("delete_message_seconds", fmt.Sprintf("%d", v.(int)))
	}

	reason := d.Get("reason").(string)
	if err := c.DoJSONWithReason(ctx, "PUT", fmt.Sprintf("/guilds/%s/bans/%s", serverID, userID), q, nil, nil, reason); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(banID(serverID, userID))
	return resourceDiscordBanRead(ctx, d, m)
}

func resourceDiscordBanRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	serverID, userID, err := parseTwoIds(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var out interface{}
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/bans/%s", serverID, userID), nil, nil, &out); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("server_id", serverID)
	_ = d.Set("user_id", userID)
	return nil
}

func resourceDiscordBanDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	serverID := d.Get("server_id").(string)
	userID := d.Get("user_id").(string)
	reason := d.Get("reason").(string)

	if err := c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/guilds/%s/bans/%s", serverID, userID), nil, nil, nil, reason); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
