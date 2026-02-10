package discord

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restUserDS struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
}

type restMemberDS struct {
	User         restUserDS `json:"user"`
	Nick         string     `json:"nick"`
	Roles        []string   `json:"roles"`
	JoinedAt     string     `json:"joined_at"`
	PremiumSince string     `json:"premium_since"`
}

func dataSourceDiscordMember() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceMemberRead,
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"user_id": {
				ExactlyOneOf: []string{"user_id", "username"},
				Type:         schema.TypeString,
				Optional:     true,
			},
			"username": {
				ExactlyOneOf: []string{"user_id", "username"},
				RequiredWith: []string{"username", "discriminator"},
				Type:         schema.TypeString,
				Optional:     true,
			},
			"discriminator": {
				RequiredWith: []string{"username", "discriminator"},
				Type:         schema.TypeString,
				Optional:     true,
			},
			"joined_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"premium_since": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"avatar": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"nick": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"roles": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Set:      schema.HashString,
			},
			"in_server": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func clearMemberState(d *schema.ResourceData) {
	_ = d.Set("joined_at", nil)
	_ = d.Set("premium_since", nil)
	_ = d.Set("roles", nil)
	_ = d.Set("username", nil)
	_ = d.Set("discriminator", nil)
	_ = d.Set("avatar", nil)
	_ = d.Set("nick", nil)
}

func dataSourceMemberRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	if _, ok := d.GetOk("username"); ok {
		// Name-based lookups require listing/searching guild members, which is not scalable and can require
		// privileged intents/permissions. IDs are the stable identifier.
		d.SetId("")
		_ = d.Set("in_server", false)
		clearMemberState(d)
		return diag.Errorf("discord_member data source lookup by username/discriminator is not supported; use user_id")
	}

	userID := d.Get("user_id").(string)
	if userID == "" {
		return diag.Errorf("either user_id or username must be set")
	}

	var member restMemberDS
	err := c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/members/"+userID, nil, nil, &member)
	if err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId(userID)
			_ = d.Set("in_server", false)
			clearMemberState(d)
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId(member.User.ID)
	_ = d.Set("in_server", true)
	_ = d.Set("joined_at", member.JoinedAt)
	if member.PremiumSince != "" {
		_ = d.Set("premium_since", member.PremiumSince)
	} else {
		_ = d.Set("premium_since", nil)
	}
	_ = d.Set("roles", member.Roles)
	_ = d.Set("username", member.User.Username)
	_ = d.Set("discriminator", member.User.Discriminator)
	_ = d.Set("avatar", member.User.Avatar)
	_ = d.Set("nick", member.Nick)
	return nil
}
