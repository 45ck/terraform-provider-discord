package discord

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDiscordRoleEveryone() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRoleEveryoneRead,
		ReadContext:   resourceRoleEveryoneRead,
		UpdateContext: resourceRoleEveryoneUpdate,
		DeleteContext: func(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
			return []diag.Diagnostic{{
				Severity: diag.Warning,
				Summary:  "Deleting the everyone role is not allowed",
			}}
		},
		Importer: &schema.ResourceImporter{
			StateContext: resourceRoleEveryoneImport,
		},
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"permissions": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"permissions_bits64": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Permissions as 64-bit integer string (decimal or 0x...). Prefer this for newer high-bit permissions.",
			},
		},
	}
}

func resourceRoleEveryoneImport(ctx context.Context, data *schema.ResourceData, i interface{}) ([]*schema.ResourceData, error) {
	// Import format is just the server/guild ID (the @everyone role has the same ID as the guild).
	data.SetId(data.Id())
	_ = data.Set("server_id", data.Id())
	return schema.ImportStatePassthroughContext(ctx, data, i)
}

func resourceRoleEveryoneRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	serverID := d.Get("server_id").(string)
	d.SetId(serverID)

	role, err := fetchRoleByID(ctx, c, serverID, serverID)
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("permissions_bits64", strings.TrimSpace(role.Permissions))
	if v, err := uint64StringToPermissionBit(role.Permissions); err == nil {
		if i, err := uint64ToIntIfFits(v); err == nil {
			_ = d.Set("permissions", i)
		} else {
			_ = d.Set("permissions", 0)
		}
	}

	return nil
}

func resourceRoleEveryoneUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	serverID := d.Get("server_id").(string)
	d.SetId(serverID)

	perms := uint64(d.Get("permissions").(int))
	if s := strings.TrimSpace(d.Get("permissions_bits64").(string)); s != "" {
		v, err := uint64StringToPermissionBit(s)
		if err != nil {
			return diag.Errorf("invalid permissions_bits64: %s", err.Error())
		}
		perms = v
	}

	body := restRoleUpdate{
		Permissions: strconv.FormatUint(perms, 10),
	}

	var out restRoleFull
	if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s/roles/%s", serverID, serverID), nil, body, &out); err != nil {
		return diag.FromErr(err)
	}

	return resourceRoleEveryoneRead(ctx, d, m)
}
