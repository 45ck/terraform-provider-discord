package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restChannel struct {
	ID                     string               `json:"id"`
	GuildID                string               `json:"guild_id"`
	Name                   string               `json:"name"`
	Type                   uint                 `json:"type"`
	Position               int                  `json:"position"`
	ParentID               string               `json:"parent_id"`
	Topic                  string               `json:"topic"`
	NSFW                   bool                 `json:"nsfw"`
	RateLimitPerUser       int                  `json:"rate_limit_per_user"`
	Bitrate                int                  `json:"bitrate"`
	UserLimit              int                  `json:"user_limit"`
	RTCRegion              string               `json:"rtc_region"`
	VideoQualityMode       int                  `json:"video_quality_mode"`
	DefaultAutoArchiveDur  int                  `json:"default_auto_archive_duration"`
	DefaultThreadRateLimit int                  `json:"default_thread_rate_limit_per_user"`
	AvailableTags          []restForumTag       `json:"available_tags"`
	DefaultReactionEmoji   *restDefaultReaction `json:"default_reaction_emoji"`
	DefaultSortOrder       int                  `json:"default_sort_order"`
	DefaultForumLayout     int                  `json:"default_forum_layout"`
}

type restForumTag struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Moderated bool   `json:"moderated"`
	EmojiID   string `json:"emoji_id"`
	EmojiName string `json:"emoji_name"`
}

type restDefaultReaction struct {
	EmojiID   string `json:"emoji_id"`
	EmojiName string `json:"emoji_name"`
}

func resourceDiscordChannel() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordChannelCreate,
		ReadContext:   resourceDiscordChannelRead,
		UpdateContext: resourceDiscordChannelUpdate,
		DeleteContext: resourceDiscordChannelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"position": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"parent_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"topic": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"nsfw": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"rate_limit_per_user": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"bitrate": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"user_limit": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"rtc_region": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"video_quality_mode": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"default_auto_archive_duration": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"default_thread_rate_limit_per_user": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"available_tag": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":         {Type: schema.TypeString, Optional: true},
						"name":       {Type: schema.TypeString, Required: true},
						"moderated":  {Type: schema.TypeBool, Optional: true},
						"emoji_id":   {Type: schema.TypeString, Optional: true},
						"emoji_name": {Type: schema.TypeString, Optional: true},
					},
				},
				Description: "Forum/media available tags.",
			},
			"default_reaction_emoji": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"emoji_id":   {Type: schema.TypeString, Optional: true},
						"emoji_name": {Type: schema.TypeString, Optional: true},
					},
				},
				Description: "Forum default reaction emoji.",
			},
			"default_sort_order": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Forum default sort order.",
			},
			"default_forum_layout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Forum default layout.",
			},
		},
	}
}

func expandForumTags(v []interface{}) []restForumTag {
	out := make([]restForumTag, 0, len(v))
	for _, raw := range v {
		m := raw.(map[string]interface{})
		id, _ := m["id"].(string)
		moderated := false
		if vv, ok := m["moderated"]; ok && vv != nil {
			moderated = vv.(bool)
		}
		tag := restForumTag{
			ID:        id,
			Name:      m["name"].(string),
			Moderated: moderated,
		}
		if s, ok := m["emoji_id"].(string); ok && s != "" {
			tag.EmojiID = s
		}
		if s, ok := m["emoji_name"].(string); ok && s != "" {
			tag.EmojiName = s
		}
		out = append(out, tag)
	}
	return out
}

func flattenForumTags(v []restForumTag) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(v))
	for _, tag := range v {
		out = append(out, map[string]interface{}{
			"id":         tag.ID,
			"name":       tag.Name,
			"moderated":  tag.Moderated,
			"emoji_id":   tag.EmojiID,
			"emoji_name": tag.EmojiName,
		})
	}
	return out
}

func validateChannelType(t string) (uint, error) {
	v, ok := getDiscordChannelType(t)
	if !ok {
		return 0, fmt.Errorf("unsupported channel type: %s", t)
	}
	// Avoid letting users create channels that are generally not creatable via this resource.
	switch t {
	case "announcement_thread", "public_thread", "private_thread":
		return 0, fmt.Errorf("thread types are not created via discord_channel; use discord_thread instead")
	}
	return v, nil
}

func resourceDiscordChannelCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	var diags diag.Diagnostics

	typ, err := validateChannelType(d.Get("type").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	body := map[string]interface{}{
		"type": int(typ),
		"name": d.Get("name").(string),
	}
	// Optional fields. This tries to avoid sending empty/default values that would
	// overwrite server defaults, but isn't perfect (Terraform SDK doesn't reliably
	// tell "unset" vs "explicit zero" for scalars without additional patterns).
	if v, ok := d.GetOk("position"); ok {
		body["position"] = v.(int)
	}
	if v, ok := d.GetOk("topic"); ok && v.(string) != "" {
		body["topic"] = v.(string)
	}
	if v, ok := d.GetOk("nsfw"); ok {
		body["nsfw"] = v.(bool)
	}
	if v, ok := d.GetOk("rate_limit_per_user"); ok {
		body["rate_limit_per_user"] = v.(int)
	}
	if v, ok := d.GetOk("bitrate"); ok {
		body["bitrate"] = v.(int)
	}
	if v, ok := d.GetOk("user_limit"); ok {
		body["user_limit"] = v.(int)
	}
	if v, ok := d.GetOk("rtc_region"); ok && v.(string) != "" {
		body["rtc_region"] = v.(string)
	}
	if v, ok := d.GetOk("video_quality_mode"); ok {
		body["video_quality_mode"] = v.(int)
	}
	if v, ok := d.GetOk("default_auto_archive_duration"); ok {
		body["default_auto_archive_duration"] = v.(int)
	}
	if v, ok := d.GetOk("default_thread_rate_limit_per_user"); ok {
		body["default_thread_rate_limit_per_user"] = v.(int)
	}
	if v, ok := d.GetOk("available_tag"); ok {
		body["available_tags"] = expandForumTags(v.([]interface{}))
	}
	if v, ok := d.GetOk("default_reaction_emoji"); ok {
		list := v.([]interface{})
		if len(list) > 0 {
			rm := list[0].(map[string]interface{})
			body["default_reaction_emoji"] = map[string]interface{}{
				"emoji_id":   rm["emoji_id"].(string),
				"emoji_name": rm["emoji_name"].(string),
			}
		}
	}
	if v, ok := d.GetOk("default_sort_order"); ok {
		body["default_sort_order"] = v.(int)
	}
	if v, ok := d.GetOk("default_forum_layout"); ok {
		body["default_forum_layout"] = v.(int)
	}
	if v, ok := d.GetOk("parent_id"); ok && v.(string) != "" {
		body["parent_id"] = v.(string)
	}

	var out restChannel
	path := fmt.Sprintf("/guilds/%s/channels", d.Get("server_id").(string))
	if err := c.DoJSON(ctx, "POST", path, nil, body, &out); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(out.ID)
	return append(diags, resourceDiscordChannelRead(ctx, d, m)...)
}

func resourceDiscordChannelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	var out restChannel
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/channels/%s", d.Id()), nil, nil, &out); err != nil {
		// If channel is gone, let Terraform recreate.
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	t, ok := getTextChannelType(out.Type)
	if ok {
		_ = d.Set("type", t)
	} else {
		_ = d.Set("type", fmt.Sprintf("%d", out.Type))
	}

	_ = d.Set("server_id", out.GuildID)
	_ = d.Set("name", out.Name)
	_ = d.Set("position", out.Position)
	if out.ParentID != "" {
		_ = d.Set("parent_id", out.ParentID)
	} else {
		_ = d.Set("parent_id", nil)
	}
	_ = d.Set("topic", out.Topic)
	_ = d.Set("nsfw", out.NSFW)
	_ = d.Set("rate_limit_per_user", out.RateLimitPerUser)
	_ = d.Set("bitrate", out.Bitrate)
	_ = d.Set("user_limit", out.UserLimit)
	_ = d.Set("rtc_region", out.RTCRegion)
	_ = d.Set("video_quality_mode", out.VideoQualityMode)
	_ = d.Set("default_auto_archive_duration", out.DefaultAutoArchiveDur)
	_ = d.Set("default_thread_rate_limit_per_user", out.DefaultThreadRateLimit)
	if out.AvailableTags != nil {
		_ = d.Set("available_tag", flattenForumTags(out.AvailableTags))
	} else {
		_ = d.Set("available_tag", nil)
	}
	if out.DefaultReactionEmoji != nil {
		_ = d.Set("default_reaction_emoji", []map[string]interface{}{{
			"emoji_id":   out.DefaultReactionEmoji.EmojiID,
			"emoji_name": out.DefaultReactionEmoji.EmojiName,
		}})
	} else {
		_ = d.Set("default_reaction_emoji", nil)
	}
	_ = d.Set("default_sort_order", out.DefaultSortOrder)
	_ = d.Set("default_forum_layout", out.DefaultForumLayout)

	return nil
}

func resourceDiscordChannelUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	body := map[string]interface{}{}

	if d.HasChange("name") {
		body["name"] = d.Get("name").(string)
	}
	if d.HasChange("position") {
		body["position"] = d.Get("position").(int)
	}
	if d.HasChange("parent_id") {
		v := d.Get("parent_id").(string)
		if v == "" {
			body["parent_id"] = nil
		} else {
			body["parent_id"] = v
		}
	}
	for _, k := range []string{
		"topic",
		"nsfw",
		"rate_limit_per_user",
		"bitrate",
		"user_limit",
		"rtc_region",
		"video_quality_mode",
		"default_auto_archive_duration",
		"default_thread_rate_limit_per_user",
	} {
		if d.HasChange(k) {
			body[k] = d.Get(k)
		}
	}
	if d.HasChange("available_tag") {
		v := d.Get("available_tag").([]interface{})
		body["available_tags"] = expandForumTags(v)
	}
	if d.HasChange("default_reaction_emoji") {
		v := d.Get("default_reaction_emoji").([]interface{})
		if len(v) == 0 {
			body["default_reaction_emoji"] = nil
		} else {
			rm := v[0].(map[string]interface{})
			body["default_reaction_emoji"] = map[string]interface{}{
				"emoji_id":   rm["emoji_id"].(string),
				"emoji_name": rm["emoji_name"].(string),
			}
		}
	}
	if d.HasChange("default_sort_order") {
		body["default_sort_order"] = d.Get("default_sort_order").(int)
	}
	if d.HasChange("default_forum_layout") {
		body["default_forum_layout"] = d.Get("default_forum_layout").(int)
	}

	if len(body) == 0 {
		return nil
	}

	var out restChannel
	if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/channels/%s", d.Id()), nil, body, &out); err != nil {
		return diag.FromErr(err)
	}

	return resourceDiscordChannelRead(ctx, d, m)
}

func resourceDiscordChannelDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	if err := c.DoJSON(ctx, "DELETE", fmt.Sprintf("/channels/%s", d.Id()), nil, nil, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
