package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restEmoji struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Roles    []string `json:"roles"`
	Managed  bool     `json:"managed"`
	Animated bool     `json:"animated"`
}

func resourceDiscordEmoji() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordEmojiCreate,
		ReadContext:   resourceDiscordEmojiRead,
		UpdateContext: resourceDiscordEmojiUpdate,
		DeleteContext: resourceDiscordEmojiDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"image_data_uri": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "data: URI for the emoji image",
			},
			"roles": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"managed": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"animated": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func resourceDiscordEmojiCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	roles := []string{}
	if v, ok := d.GetOk("roles"); ok {
		for _, r := range v.(*schema.Set).List() {
			roles = append(roles, r.(string))
		}
	}

	body := map[string]interface{}{
		"name":  d.Get("name").(string),
		"image": d.Get("image_data_uri").(string),
		"roles": roles,
	}

	var out restEmoji
	if err := c.DoJSON(ctx, "POST", fmt.Sprintf("/guilds/%s/emojis", serverID), nil, body, &out); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(out.ID)
	return resourceDiscordEmojiRead(ctx, d, m)
}

func resourceDiscordEmojiRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var out restEmoji
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/emojis/%s", serverID, d.Id()), nil, nil, &out); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", out.Name)
	_ = d.Set("roles", out.Roles)
	_ = d.Set("managed", out.Managed)
	_ = d.Set("animated", out.Animated)
	return nil
}

func resourceDiscordEmojiUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	roles := []string{}
	if v, ok := d.GetOk("roles"); ok {
		for _, r := range v.(*schema.Set).List() {
			roles = append(roles, r.(string))
		}
	}

	body := map[string]interface{}{
		"name":  d.Get("name").(string),
		"roles": roles,
	}

	var out restEmoji
	if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s/emojis/%s", serverID, d.Id()), nil, body, &out); err != nil {
		return diag.FromErr(err)
	}

	return resourceDiscordEmojiRead(ctx, d, m)
}

func resourceDiscordEmojiDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	if err := c.DoJSON(ctx, "DELETE", fmt.Sprintf("/guilds/%s/emojis/%s", serverID, d.Id()), nil, nil, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
