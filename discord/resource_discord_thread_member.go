package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restThreadMember struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	JoinTimestamp string `json:"join_timestamp"`
	Flags         int    `json:"flags"`
}

// discord_thread_member manages a member's presence in a thread.
// Use user_id = "@me" to manage the bot's membership.
func resourceDiscordThreadMember() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordThreadMemberCreate,
		ReadContext:   resourceDiscordThreadMemberRead,
		DeleteContext: resourceDiscordThreadMemberDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"thread_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Thread channel ID.",
			},
			"user_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "User ID, or @me for the bot.",
			},
			"reason": {
				Type:     schema.TypeString,
				Optional: true,
				// Not readable
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool { return true },
			},
			"join_timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"flags": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func threadMemberID(threadID, userID string) string {
	return fmt.Sprintf("%s:%s", threadID, userID)
}

func threadMemberPath(threadID, userID string) string {
	if userID == "@me" {
		return fmt.Sprintf("/channels/%s/thread-members/@me", threadID)
	}
	return fmt.Sprintf("/channels/%s/thread-members/%s", threadID, userID)
}

func resourceDiscordThreadMemberCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	threadID := d.Get("thread_id").(string)
	userID := d.Get("user_id").(string)
	reason := d.Get("reason").(string)

	// PUT add thread member. API usually returns 204 for @me and 204/200 for others.
	if err := c.DoJSONWithReason(ctx, "PUT", threadMemberPath(threadID, userID), nil, nil, nil, reason); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(threadMemberID(threadID, userID))
	return resourceDiscordThreadMemberRead(ctx, d, m)
}

func resourceDiscordThreadMemberRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	threadID, userID, err := parseTwoIds(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var out restThreadMember
	if err := c.DoJSON(ctx, "GET", threadMemberPath(threadID, userID), nil, nil, &out); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("thread_id", threadID)
	_ = d.Set("user_id", userID)
	_ = d.Set("join_timestamp", out.JoinTimestamp)
	_ = d.Set("flags", out.Flags)
	return nil
}

func resourceDiscordThreadMemberDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	threadID := d.Get("thread_id").(string)
	userID := d.Get("user_id").(string)
	reason := d.Get("reason").(string)

	if err := c.DoJSONWithReason(ctx, "DELETE", threadMemberPath(threadID, userID), nil, nil, nil, reason); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
