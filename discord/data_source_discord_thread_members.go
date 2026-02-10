package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/url"
	"strconv"
)

func dataSourceDiscordThreadMembers() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDiscordThreadMembersRead,
		Schema: map[string]*schema.Schema{
			"thread_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"limit": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"after": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Snowflake user ID; return thread members after this user.",
			},
			"with_member": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Include guild member object where supported.",
			},
			"member": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"user_id": {
							Type:     schema.TypeString,
							Computed: true,
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
				},
			},
		},
	}
}

func dataSourceDiscordThreadMembersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	threadID := d.Get("thread_id").(string)

	q := url.Values{}
	if v, ok := d.GetOk("limit"); ok {
		q.Set("limit", strconv.Itoa(v.(int)))
	}
	if v, ok := d.GetOk("after"); ok && v.(string) != "" {
		q.Set("after", v.(string))
	}
	if v, ok := d.GetOk("with_member"); ok {
		if v.(bool) {
			q.Set("with_member", "true")
		} else {
			q.Set("with_member", "false")
		}
	}

	var out []restThreadMember
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/channels/%s/thread-members", threadID), q, nil, &out); err != nil {
		return diag.FromErr(err)
	}

	members := make([]map[string]interface{}, 0, len(out))
	for _, tm := range out {
		uid := tm.UserID
		if uid == "" {
			uid = tm.ID
		}
		members = append(members, map[string]interface{}{
			"user_id":        uid,
			"join_timestamp": tm.JoinTimestamp,
			"flags":          tm.Flags,
		})
	}

	d.SetId(threadID)
	_ = d.Set("member", members)
	return nil
}
