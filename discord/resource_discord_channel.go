package discord

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restChannelOverwriteLegacy struct {
	ID    string `json:"id"`
	Type  int    `json:"type"`
	Allow string `json:"allow"`
	Deny  string `json:"deny"`
}

type restChannelLegacy struct {
	ID                   string                       `json:"id"`
	GuildID              string                       `json:"guild_id"`
	Name                 string                       `json:"name"`
	Type                 uint                         `json:"type"`
	Position             int                          `json:"position"`
	ParentID             string                       `json:"parent_id"`
	Topic                string                       `json:"topic"`
	NSFW                 bool                         `json:"nsfw"`
	Bitrate              int                          `json:"bitrate"`
	UserLimit            int                          `json:"user_limit"`
	PermissionOverwrites []restChannelOverwriteLegacy `json:"permission_overwrites"`
}

type restCreateGuildChannelLegacy struct {
	Name      string `json:"name"`
	Type      uint   `json:"type"`
	Topic     string `json:"topic,omitempty"`
	Bitrate   int    `json:"bitrate,omitempty"`
	UserLimit int    `json:"user_limit,omitempty"`
	ParentID  string `json:"parent_id,omitempty"`
	NSFW      bool   `json:"nsfw,omitempty"`
	Position  int    `json:"position,omitempty"`
}

type restModifyChannelLegacy struct {
	Name            *string `json:"name,omitempty"`
	Position        *int    `json:"position,omitempty"`
	ParentID        *string `json:"parent_id,omitempty"`
	LockPermissions *bool   `json:"lock_permissions,omitempty"`
	Topic           *string `json:"topic,omitempty"`
	NSFW            *bool   `json:"nsfw,omitempty"`
	Bitrate         *int    `json:"bitrate,omitempty"`
	UserLimit       *int    `json:"user_limit,omitempty"`
}

func getChannelSchema(channelType string, s map[string]*schema.Schema) map[string]*schema.Schema {
	addedSchema := map[string]*schema.Schema{
		"server_id": {
			Type:     schema.TypeString,
			Required: true,
		},
		"type": {
			Type:     schema.TypeString,
			Required: true,
			ValidateDiagFunc: func(i interface{}, path cty.Path) (diags diag.Diagnostics) {
				if i.(string) != channelType {
					diags = append(diags, diag.Errorf("type must be %s, %s passed", channelType, i.(string))...)
				}

				return diags
			},
			DefaultFunc: func() (interface{}, error) {
				return channelType, nil
			},
		},
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"position": {
			Type:     schema.TypeInt,
			Default:  1,
			Optional: true,
		},
	}

	if channelType != "category" {
		addedSchema["category"] = &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		}
		addedSchema["sync_perms_with_category"] = &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
		}
	}

	if s != nil {
		for k, v := range s {
			addedSchema[k] = v
		}
	}

	return addedSchema
}

func validateChannel(d *schema.ResourceData) (bool, error) {
	channelType := d.Get("type").(string)

	if channelType == "category" {
		if _, ok := d.GetOk("category"); ok {
			return false, errors.New("category cannot be a child of another category")
		}
		if _, ok := d.GetOk("nsfw"); ok {
			return false, errors.New("nsfw is not allowed on categories")
		}
	}

	if channelType == "voice" {
		if _, ok := d.GetOk("topic"); ok {
			return false, errors.New("topic is not allowed on voice channels")
		}
		if _, ok := d.GetOk("nsfw"); ok {
			return false, errors.New("nsfw is not allowed on voice channels")
		}
	}

	if channelType == "text" {
		if _, ok := d.GetOk("bitrate"); ok {
			return false, errors.New("bitrate is not allowed on text channels")
		}
		if _, ok := d.GetOk("user_limit"); ok {
			if d.Get("user_limit").(int) > 0 {
				return false, errors.New("user_limit is not allowed on text channels")
			}
		}
		name := d.Get("name").(string)
		if strings.ToLower(name) != name {
			return false, errors.New("name must be lowercase")
		}
	}

	return true, nil
}

func overwritesEqual(a, b []restChannelOverwriteLegacy) bool {
	if len(a) != len(b) {
		return false
	}

	index := map[string]restChannelOverwriteLegacy{}
	for _, x := range a {
		index[fmt.Sprintf("%d:%s", x.Type, x.ID)] = x
	}
	for _, x := range b {
		k := fmt.Sprintf("%d:%s", x.Type, x.ID)
		v, ok := index[k]
		if !ok {
			return false
		}
		if v.Allow != x.Allow || v.Deny != x.Deny {
			return false
		}
	}
	return true
}

func resourceChannelCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	if ok, reason := validateChannel(d); !ok {
		return diag.FromErr(reason)
	}

	serverID := d.Get("server_id").(string)
	channelType := d.Get("type").(string)
	channelTypeInt, ok := getDiscordChannelType(channelType)
	if !ok {
		return diag.Errorf("invalid channel type: %s", channelType)
	}

	body := restCreateGuildChannelLegacy{
		Name:     d.Get("name").(string),
		Type:     channelTypeInt,
		Position: d.Get("position").(int),
	}

	if channelType == "text" {
		if v, ok := d.GetOk("topic"); ok {
			body.Topic = v.(string)
		}
		if v, ok := d.GetOk("nsfw"); ok {
			body.NSFW = v.(bool)
		}
	} else if channelType == "voice" {
		if v, ok := d.GetOk("bitrate"); ok {
			body.Bitrate = v.(int)
		}
		if v, ok := d.GetOk("user_limit"); ok {
			body.UserLimit = v.(int)
		}
	}

	if channelType != "category" {
		if v, ok := d.GetOk("category"); ok {
			body.ParentID = v.(string)
		}
	}

	var out restChannelLegacy
	if err := c.DoJSON(ctx, "POST", fmt.Sprintf("/guilds/%s/channels", serverID), nil, body, &out); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(out.ID)
	_ = d.Set("server_id", serverID)

	if channelType != "category" {
		if v, ok := d.GetOk("sync_perms_with_category"); ok && v.(bool) {
			if out.ParentID == "" {
				return diag.Errorf("can't sync permissions with category: channel (%s) doesn't have a category", out.ID)
			}

			// Use lock_permissions to sync overwrites with the parent category.
			parentID := out.ParentID
			lock := true
			mod := restModifyChannelLegacy{
				ParentID:        &parentID,
				LockPermissions: &lock,
			}
			if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/channels/%s", out.ID), nil, mod, nil); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return resourceChannelRead(ctx, d, m)
}

func resourceChannelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	var ch restChannelLegacy
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/channels/%s", d.Id()), nil, nil, &ch); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	channelType, ok := getTextChannelType(ch.Type)
	if !ok {
		return diag.Errorf("invalid channel type: %d", ch.Type)
	}

	_ = d.Set("type", channelType)
	_ = d.Set("name", ch.Name)
	_ = d.Set("position", ch.Position)

	if channelType == "text" {
		_ = d.Set("topic", ch.Topic)
		_ = d.Set("nsfw", ch.NSFW)
	} else if channelType == "voice" {
		_ = d.Set("bitrate", ch.Bitrate)
		_ = d.Set("user_limit", ch.UserLimit)
	}

	if channelType != "category" {
		if ch.ParentID != "" {
			var parent restChannelLegacy
			if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/channels/%s", ch.ParentID), nil, nil, &parent); err != nil {
				return diag.FromErr(err)
			}
			_ = d.Set("sync_perms_with_category", overwritesEqual(ch.PermissionOverwrites, parent.PermissionOverwrites))
		} else {
			_ = d.Set("sync_perms_with_category", false)
		}
	}

	if ch.ParentID != "" {
		_ = d.Set("category", ch.ParentID)
	} else {
		_ = d.Set("category", nil)
	}

	return nil
}

func resourceChannelUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	if ok, reason := validateChannel(d); !ok {
		return diag.FromErr(reason)
	}

	channelID := d.Id()
	channelType := d.Get("type").(string)

	mod := restModifyChannelLegacy{}
	any := false

	if d.HasChange("name") {
		v := d.Get("name").(string)
		mod.Name = &v
		any = true
	}
	if d.HasChange("position") {
		v := d.Get("position").(int)
		mod.Position = &v
		any = true
	}

	if channelType == "text" {
		if d.HasChange("topic") {
			v := d.Get("topic").(string)
			mod.Topic = &v
			any = true
		}
		if d.HasChange("nsfw") {
			v := d.Get("nsfw").(bool)
			mod.NSFW = &v
			any = true
		}
	} else if channelType == "voice" {
		if d.HasChange("bitrate") {
			v := d.Get("bitrate").(int)
			mod.Bitrate = &v
			any = true
		}
		if d.HasChange("user_limit") {
			v := d.Get("user_limit").(int)
			mod.UserLimit = &v
			any = true
		}
	}

	if channelType != "category" {
		if d.HasChange("category") {
			v := d.Get("category").(string)
			if v == "" {
				mod.ParentID = nil
			} else {
				mod.ParentID = &v
			}
			any = true
		}
		if d.HasChange("sync_perms_with_category") || d.HasChange("category") {
			if d.Get("sync_perms_with_category").(bool) && d.Get("category").(string) != "" {
				lock := true
				mod.LockPermissions = &lock
				any = true
			}
		}
	}

	if any {
		if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/channels/%s", channelID), nil, mod, nil); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceChannelRead(ctx, d, m)
}

func resourceChannelDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	if err := c.DoJSON(ctx, "DELETE", fmt.Sprintf("/channels/%s", d.Id()), nil, nil, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
