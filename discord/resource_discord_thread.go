package discord

import (
	"context"
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restThreadMetadata struct {
	Archived       bool `json:"archived"`
	AutoArchiveDur int  `json:"auto_archive_duration"`
	Locked         bool `json:"locked"`
	Invitable      bool `json:"invitable"`
}

type restThreadChannel struct {
	ID             string              `json:"id"`
	GuildID        string              `json:"guild_id"`
	ParentID       string              `json:"parent_id"`
	Name           string              `json:"name"`
	Type           uint                `json:"type"`
	RateLimit      int                 `json:"rate_limit_per_user"`
	ThreadMetadata *restThreadMetadata `json:"thread_metadata"`
	AppliedTags    []string            `json:"applied_tags"`
}

func resourceDiscordThread() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordThreadCreate,
		ReadContext:   resourceDiscordThreadRead,
		UpdateContext: resourceDiscordThreadUpdate,
		DeleteContext: resourceDiscordThreadDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"channel_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Parent channel ID (text/news/forum/media).",
			},
			"message_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "If set, starts a thread from an existing message.",
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "public_thread",
				Description: "Thread type: public_thread, private_thread, announcement_thread.",
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"auto_archive_duration": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Minutes: 60, 1440, 4320, 10080.",
			},
			"invitable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Private thread invite permission.",
			},
			"rate_limit_per_user": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Thread slowmode in seconds.",
			},
			"archived": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether the thread is archived.",
			},
			"locked": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether the thread is locked.",
			},
			"applied_tags": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
				Description: "Forum/media thread tags (IDs).",
			},
			"server_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			// Optional initial message for forum/media threads.
			"content": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Initial message content (forum/media threads).",
			},
			"embed": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: embedSchema(),
				},
			},
		},
	}
}

// embedSchema matches the single-embed schema used by discord_message.
func embedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"title":       {Type: schema.TypeString, Optional: true},
		"description": {Type: schema.TypeString, Optional: true},
		"url":         {Type: schema.TypeString, Optional: true},
		"timestamp":   {Type: schema.TypeString, Optional: true},
		"color":       {Type: schema.TypeInt, Optional: true},
		"footer": {
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"text":     {Type: schema.TypeString, Required: true},
				"icon_url": {Type: schema.TypeString, Optional: true},
			}},
		},
		"image": {
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"url":       {Type: schema.TypeString, Required: true},
				"proxy_url": {Type: schema.TypeString, Computed: true},
				"height":    {Type: schema.TypeInt, Optional: true},
				"width":     {Type: schema.TypeInt, Optional: true},
			}},
		},
		"thumbnail": {
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"url":       {Type: schema.TypeString, Required: true},
				"proxy_url": {Type: schema.TypeString, Computed: true},
				"height":    {Type: schema.TypeInt, Optional: true},
				"width":     {Type: schema.TypeInt, Optional: true},
			}},
		},
		"video": {
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"url":    {Type: schema.TypeString, Required: true},
				"height": {Type: schema.TypeInt, Optional: true},
				"width":  {Type: schema.TypeInt, Optional: true},
			}},
		},
		"provider": {
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"name": {Type: schema.TypeString, Optional: true},
				"url":  {Type: schema.TypeString, Optional: true},
			}},
		},
		"author": {
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"name":           {Type: schema.TypeString, Optional: true},
				"url":            {Type: schema.TypeString, Optional: true},
				"icon_url":       {Type: schema.TypeString, Optional: true},
				"proxy_icon_url": {Type: schema.TypeString, Computed: true},
			}},
		},
		"fields": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"name":   {Type: schema.TypeString, Required: true},
				"value":  {Type: schema.TypeString, Optional: true},
				"inline": {Type: schema.TypeBool, Optional: true},
			}},
		},
	}
}

func threadTypeToInt(t string) (int, error) {
	v, ok := getDiscordChannelType(t)
	if !ok {
		return 0, fmt.Errorf("unsupported thread type: %s", t)
	}
	switch t {
	case "announcement_thread", "public_thread", "private_thread":
		return int(v), nil
	default:
		return 0, fmt.Errorf("unsupported thread type: %s", t)
	}
}

func resourceDiscordThreadCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	parentID := d.Get("channel_id").(string)

	typ, err := threadTypeToInt(d.Get("type").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	body := map[string]interface{}{
		"name": d.Get("name").(string),
		"type": typ,
	}
	if v, ok := d.GetOk("auto_archive_duration"); ok {
		body["auto_archive_duration"] = v.(int)
	}
	if v, ok := d.GetOk("invitable"); ok {
		body["invitable"] = v.(bool)
	}
	if v, ok := d.GetOk("rate_limit_per_user"); ok {
		body["rate_limit_per_user"] = v.(int)
	}
	if v, ok := d.GetOk("applied_tags"); ok {
		tags := []string{}
		for _, x := range v.(*schema.Set).List() {
			tags = append(tags, x.(string))
		}
		body["applied_tags"] = tags
	}

	// Optional initial message (forum/media).
	if v, ok := d.GetOk("content"); ok {
		msg := map[string]interface{}{"content": v.(string)}
		if ev, ok := d.GetOk("embed"); ok {
			embed, err := buildEmbed(ev.([]interface{}))
			if err != nil {
				return diag.FromErr(err)
			}
			// API expects embeds array.
			msg["embeds"] = []*disgord.Embed{embed}
		}
		body["message"] = msg
	} else if ev, ok := d.GetOk("embed"); ok {
		embed, err := buildEmbed(ev.([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		}
		body["message"] = map[string]interface{}{"embeds": []*disgord.Embed{embed}}
	}

	var out restThreadChannel
	if msgID, ok := d.GetOk("message_id"); ok && msgID.(string) != "" {
		path := fmt.Sprintf("/channels/%s/messages/%s/threads", parentID, msgID.(string))
		if err := c.DoJSON(ctx, "POST", path, nil, body, &out); err != nil {
			return diag.FromErr(err)
		}
	} else {
		path := fmt.Sprintf("/channels/%s/threads", parentID)
		if err := c.DoJSON(ctx, "POST", path, nil, body, &out); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(out.ID)
	return resourceDiscordThreadRead(ctx, d, m)
}

func resourceDiscordThreadRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	var out restThreadChannel
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/channels/%s", d.Id()), nil, nil, &out); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	tt, ok := getTextChannelType(out.Type)
	if ok {
		_ = d.Set("type", tt)
	}

	_ = d.Set("server_id", out.GuildID)
	_ = d.Set("channel_id", out.ParentID)
	_ = d.Set("name", out.Name)
	_ = d.Set("rate_limit_per_user", out.RateLimit)

	if out.ThreadMetadata != nil {
		_ = d.Set("archived", out.ThreadMetadata.Archived)
		_ = d.Set("locked", out.ThreadMetadata.Locked)
		_ = d.Set("invitable", out.ThreadMetadata.Invitable)
		_ = d.Set("auto_archive_duration", out.ThreadMetadata.AutoArchiveDur)
	}

	if out.AppliedTags != nil {
		_ = d.Set("applied_tags", out.AppliedTags)
	} else {
		_ = d.Set("applied_tags", nil)
	}

	return nil
}

func resourceDiscordThreadUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	body := map[string]interface{}{}
	if d.HasChange("name") {
		body["name"] = d.Get("name").(string)
	}
	if d.HasChange("rate_limit_per_user") {
		body["rate_limit_per_user"] = d.Get("rate_limit_per_user").(int)
	}
	if d.HasChange("archived") {
		body["archived"] = d.Get("archived").(bool)
	}
	if d.HasChange("locked") {
		body["locked"] = d.Get("locked").(bool)
	}
	if d.HasChange("auto_archive_duration") {
		body["auto_archive_duration"] = d.Get("auto_archive_duration").(int)
	}
	if d.HasChange("invitable") {
		body["invitable"] = d.Get("invitable").(bool)
	}
	if d.HasChange("applied_tags") {
		tags := []string{}
		if v, ok := d.GetOk("applied_tags"); ok {
			for _, x := range v.(*schema.Set).List() {
				tags = append(tags, x.(string))
			}
		}
		body["applied_tags"] = tags
	}

	if len(body) == 0 {
		return nil
	}

	if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/channels/%s", d.Id()), nil, body, nil); err != nil {
		return diag.FromErr(err)
	}
	return resourceDiscordThreadRead(ctx, d, m)
}

func resourceDiscordThreadDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	// Threads are channels; delete is a normal channel delete.
	if err := c.DoJSON(ctx, "DELETE", fmt.Sprintf("/channels/%s", d.Id()), nil, nil, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
