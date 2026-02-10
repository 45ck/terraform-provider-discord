package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDiscordMemberNickname() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordMemberNicknameUpsert,
		ReadContext:   resourceDiscordMemberNicknameRead,
		UpdateContext: resourceDiscordMemberNicknameUpsert,
		DeleteContext: resourceDiscordMemberNicknameDelete,
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
			"nick": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Nickname for the member. Use an empty string to clear.",
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

func resourceDiscordMemberNicknameUpsert(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	userID := d.Get("user_id").(string)
	reason := d.Get("reason").(string)

	nick := d.Get("nick").(string)
	var val interface{}
	if nick == "" {
		val = nil
	} else {
		val = nick
	}
	body := map[string]interface{}{"nick": val}

	if err := c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/members/%s", serverID, userID), nil, body, nil, reason); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s:%s", serverID, userID))
	return resourceDiscordMemberNicknameRead(ctx, d, m)
}

func resourceDiscordMemberNicknameRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
	_ = d.Set("nick", out.Nick)
	return nil
}

func resourceDiscordMemberNicknameDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	userID := d.Get("user_id").(string)
	reason := d.Get("reason").(string)

	body := map[string]interface{}{"nick": nil}
	if err := c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/members/%s", serverID, userID), nil, body, nil, reason); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
