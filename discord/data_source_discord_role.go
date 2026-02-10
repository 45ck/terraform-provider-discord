package discord

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restRoleDS struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Hoist       bool   `json:"hoist"`
	Mentionable bool   `json:"mentionable"`
	Managed     bool   `json:"managed"`
	Position    int    `json:"position"`
	// Discord returns permissions as a stringified 64-bit integer.
	Permissions string `json:"permissions"`
}

func dataSourceDiscordRole() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDiscordRoleRead,
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"role_id": {
				ExactlyOneOf: []string{"role_id", "name"},
				Type:         schema.TypeString,
				Optional:     true,
			},
			"name": {
				ExactlyOneOf: []string{"role_id", "name"},
				Type:         schema.TypeString,
				Optional:     true,
			},
			"position": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"color": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"permissions": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Role permission bits (platform-sized integer; can overflow on 32-bit). Prefer permissions_bits64.",
			},
			"permissions_bits64": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Permissions as 64-bit integer string (decimal). Prefer this for newer high-bit permissions.",
			},
			"hoist": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"mentionable": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"managed": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataSourceDiscordRoleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	var roles []restRoleDS
	if err := c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/roles", nil, nil, &roles); err != nil {
		return diag.FromErr(err)
	}

	var role *restRoleDS
	if v, ok := d.GetOk("role_id"); ok {
		id := v.(string)
		for i := range roles {
			if roles[i].ID == id {
				role = &roles[i]
				break
			}
		}
		if role == nil {
			return diag.Errorf("failed to fetch role %s in server %s", id, serverID)
		}
	}

	if v, ok := d.GetOk("name"); ok {
		name := v.(string)
		for i := range roles {
			if roles[i].Name == name {
				role = &roles[i]
				break
			}
		}
		if role == nil {
			return diag.Errorf("failed to fetch role %q in server %s", name, serverID)
		}
	}

	if role == nil {
		return diag.Errorf("either role_id or name must be set")
	}

	permsRaw := strings.TrimSpace(role.Permissions)
	perms64, err := strconv.ParseUint(permsRaw, 10, 64)
	if err != nil {
		// Keep the data source usable even if Discord changes the response shape.
		_ = d.Set("permissions_bits64", permsRaw)
		_ = d.Set("permissions", 0)
	} else {
		_ = d.Set("permissions_bits64", strconv.FormatUint(perms64, 10))
		if i, err := uint64ToIntIfFits(perms64); err == nil {
			_ = d.Set("permissions", i)
		} else {
			_ = d.Set("permissions", 0)
		}
	}

	d.SetId(role.ID)
	_ = d.Set("role_id", role.ID)
	_ = d.Set("name", role.Name)
	// Discord role positions are reverse-indexed with @everyone as 0; preserve API semantics.
	_ = d.Set("position", role.Position)
	_ = d.Set("color", role.Color)
	_ = d.Set("hoist", role.Hoist)
	_ = d.Set("mentionable", role.Mentionable)
	_ = d.Set("managed", role.Managed)
	return nil
}
