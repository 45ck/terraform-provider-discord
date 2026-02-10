package discord

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restInvite struct {
	Code string `json:"code"`
}

type restInviteCreate struct {
	MaxAge    int  `json:"max_age,omitempty"`
	MaxUses   int  `json:"max_uses,omitempty"`
	Temporary bool `json:"temporary,omitempty"`
	Unique    bool `json:"unique,omitempty"`
}

func resourceDiscordInvite() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceInviteCreate,
		ReadContext:   resourceInviteRead,
		DeleteContext: resourceInviteDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"channel_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"max_age": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Optional: true,
				Default:  86400,
			},
			"max_uses": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Optional: true,
			},
			"temporary": {
				Type:     schema.TypeBool,
				ForceNew: true,
				Optional: true,
			},
			"unique": {
				Type:     schema.TypeBool,
				ForceNew: true,
				Optional: true,
			},
			"code": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceInviteCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	channelID := d.Get("channel_id").(string)

	body := restInviteCreate{
		MaxAge:    d.Get("max_age").(int),
		MaxUses:   d.Get("max_uses").(int),
		Temporary: d.Get("temporary").(bool),
		Unique:    d.Get("unique").(bool),
	}

	var out restInvite
	if err := c.DoJSON(ctx, "POST", fmt.Sprintf("/channels/%s/invites", channelID), nil, body, &out); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(out.Code)
	_ = d.Set("code", out.Code)
	return nil
}

func resourceInviteRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	var out restInvite
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/invites/%s", d.Id()), nil, nil, &out); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	_ = d.Set("code", out.Code)
	return nil
}

func resourceInviteDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	if err := c.DoJSON(ctx, "DELETE", fmt.Sprintf("/invites/%s", d.Id()), nil, nil, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
