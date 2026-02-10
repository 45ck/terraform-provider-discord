package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"os"
	"path/filepath"
)

type restSticker struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	FormatType  int    `json:"format_type"`
}

func resourceDiscordSticker() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordStickerCreate,
		ReadContext:   resourceDiscordStickerRead,
		UpdateContext: resourceDiscordStickerUpdate,
		DeleteContext: resourceDiscordStickerDelete,
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
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"tags": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Sticker tags (comma-separated emoji names).",
			},
			"file_path": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Path to sticker asset (png/apng/lottie json).",
			},
			"format_type": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"reason": {
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool { return true },
			},
		},
	}
}

func resourceDiscordStickerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	reason := d.Get("reason").(string)

	p := d.Get("file_path").(string)
	b, err := os.ReadFile(p)
	if err != nil {
		return diag.FromErr(err)
	}

	fields := map[string]string{
		"name":        d.Get("name").(string),
		"description": d.Get("description").(string),
		"tags":        d.Get("tags").(string),
	}

	var out restSticker
	if err := c.DoMultipartWithReason(ctx, "POST", fmt.Sprintf("/guilds/%s/stickers", serverID), nil, fields, "file", filepath.Base(p), b, &out, reason); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(out.ID)
	return resourceDiscordStickerRead(ctx, d, m)
}

func resourceDiscordStickerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var out restSticker
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/stickers/%s", serverID, d.Id()), nil, nil, &out); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", out.Name)
	_ = d.Set("description", out.Description)
	_ = d.Set("tags", out.Tags)
	_ = d.Set("format_type", out.FormatType)
	return nil
}

func resourceDiscordStickerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	reason := d.Get("reason").(string)

	body := map[string]interface{}{}
	if d.HasChange("name") {
		body["name"] = d.Get("name").(string)
	}
	if d.HasChange("description") {
		body["description"] = d.Get("description").(string)
	}
	if d.HasChange("tags") {
		body["tags"] = d.Get("tags").(string)
	}

	if len(body) == 0 {
		return nil
	}

	var out restSticker
	if err := c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/stickers/%s", serverID, d.Id()), nil, body, &out, reason); err != nil {
		return diag.FromErr(err)
	}

	return resourceDiscordStickerRead(ctx, d, m)
}

func resourceDiscordStickerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	reason := d.Get("reason").(string)

	if err := c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/guilds/%s/stickers/%s", serverID, d.Id()), nil, nil, nil, reason); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
