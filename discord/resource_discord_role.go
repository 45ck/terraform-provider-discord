package discord

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restRoleFull struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Hoist       bool   `json:"hoist"`
	Mentionable bool   `json:"mentionable"`
	Managed     bool   `json:"managed"`
	Position    int    `json:"position"`
	Permissions string `json:"permissions"`
}

type restRoleCreate struct {
	Name        string `json:"name,omitempty"`
	Permissions string `json:"permissions,omitempty"`
	Color       int    `json:"color,omitempty"`
	Hoist       bool   `json:"hoist,omitempty"`
	Mentionable bool   `json:"mentionable,omitempty"`
}

type restRoleUpdate = restRoleCreate

func resourceDiscordRole() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRoleCreate,
		ReadContext:   resourceRoleRead,
		UpdateContext: resourceRoleUpdate,
		DeleteContext: resourceRoleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceRoleImport,
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
			"color": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"hoist": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"mentionable": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"position": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
				ValidateFunc: func(val interface{}, key string) (warns []string, errors []error) {
					v := val.(int)
					if v < 0 {
						errors = append(errors, fmt.Errorf("position must be greater than or equal to 0, got: %d", v))
					}
					return
				},
			},
			"managed": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func resourceRoleImport(ctx context.Context, data *schema.ResourceData, i interface{}) ([]*schema.ResourceData, error) {
	serverID, roleID, err := parseTwoIds(data.Id())
	if err != nil {
		return nil, err
	}
	data.SetId(roleID)
	_ = data.Set("server_id", serverID)
	return schema.ImportStatePassthroughContext(ctx, data, i)
}

func desiredPerms64FromSchema(d *schema.ResourceData) (uint64, diag.Diagnostics) {
	perms := uint64(d.Get("permissions").(int))
	if s := strings.TrimSpace(d.Get("permissions_bits64").(string)); s != "" {
		v, err := uint64StringToPermissionBit(s)
		if err != nil {
			return 0, diag.Errorf("invalid permissions_bits64: %s", err.Error())
		}
		perms = v
	}
	return perms, nil
}

func fetchRoleByID(ctx context.Context, c *RestClient, serverID, roleID string) (*restRoleFull, error) {
	var roles []restRoleFull
	if err := c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/roles", nil, nil, &roles); err != nil {
		return nil, err
	}
	for i := range roles {
		if roles[i].ID == roleID {
			return &roles[i], nil
		}
	}
	return nil, &DiscordHTTPError{Method: "GET", Path: "/guilds/" + serverID + "/roles", StatusCode: 404, Message: "role not found"}
}

func swapRolePosition(ctx context.Context, c *RestClient, serverID, roleID string, newPos int) diag.Diagnostics {
	var roles []restRoleFull
	if err := c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/roles", nil, nil, &roles); err != nil {
		return diag.FromErr(err)
	}

	var current *restRoleFull
	var occupant *restRoleFull
	for i := range roles {
		if roles[i].ID == roleID {
			current = &roles[i]
		}
		if roles[i].Position == newPos {
			occupant = &roles[i]
		}
	}
	if current == nil {
		return diag.Errorf("role %s not found in server %s", roleID, serverID)
	}
	if occupant == nil {
		return diag.Errorf("new role position is out of bounds: %d", newPos)
	}
	if occupant.ID == roleID {
		return nil
	}

	body := []restRolePosition{
		{ID: occupant.ID, Position: current.Position},
		{ID: roleID, Position: newPos},
	}
	var out []restRole
	if err := c.DoJSON(ctx, "PATCH", "/guilds/"+serverID+"/roles", nil, body, &out); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceRoleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)

	perms, diags := desiredPerms64FromSchema(d)
	if diags != nil {
		return diags
	}

	create := restRoleCreate{
		Name:        d.Get("name").(string),
		Permissions: strconv.FormatUint(perms, 10),
		Color:       d.Get("color").(int),
		Hoist:       d.Get("hoist").(bool),
		Mentionable: d.Get("mentionable").(bool),
	}

	var role restRoleFull
	if err := c.DoJSON(ctx, "POST", "/guilds/"+serverID+"/roles", nil, create, &role); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(role.ID)
	_ = d.Set("server_id", serverID)
	_ = d.Set("managed", role.Managed)
	_ = d.Set("permissions_bits64", strings.TrimSpace(role.Permissions))

	if newPos, ok := d.GetOk("position"); ok {
		diags = append(diags, swapRolePosition(ctx, c, serverID, role.ID, newPos.(int))...)
	}

	return append(diags, resourceRoleRead(ctx, d, m)...)
}

func resourceRoleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	roleID := d.Id()

	role, err := fetchRoleByID(ctx, c, serverID, roleID)
	if err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", role.Name)
	_ = d.Set("position", role.Position)
	_ = d.Set("color", role.Color)
	_ = d.Set("hoist", role.Hoist)
	_ = d.Set("mentionable", role.Mentionable)
	_ = d.Set("managed", role.Managed)
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

func resourceRoleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	roleID := d.Id()

	var diags diag.Diagnostics

	if d.HasChange("position") {
		_, newPos := d.GetChange("position")
		diags = append(diags, swapRolePosition(ctx, c, serverID, roleID, newPos.(int))...)
	}

	perms, pdiags := desiredPerms64FromSchema(d)
	if pdiags != nil {
		return append(diags, pdiags...)
	}

	update := restRoleUpdate{
		Name:        d.Get("name").(string),
		Permissions: strconv.FormatUint(perms, 10),
		Color:       d.Get("color").(int),
		Hoist:       d.Get("hoist").(bool),
		Mentionable: d.Get("mentionable").(bool),
	}

	var out restRoleFull
	if err := c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s/roles/%s", serverID, roleID), nil, update, &out); err != nil {
		return append(diags, diag.FromErr(err)...)
	}

	return append(diags, resourceRoleRead(ctx, d, m)...)
}

func resourceRoleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	roleID := d.Id()

	if err := c.DoJSON(ctx, "DELETE", fmt.Sprintf("/guilds/%s/roles/%s", serverID, roleID), nil, nil, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
