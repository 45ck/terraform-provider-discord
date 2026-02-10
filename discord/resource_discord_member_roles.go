package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type roleSchema struct {
	RoleID  string `json:"role_id"`
	HasRole bool   `json:"has_role"`
}

type restMemberRoles struct {
	Roles []string `json:"roles"`
}

type restMemberForRoles struct {
	Roles []string `json:"roles"`
}

func convertToRoleSchema(v interface{}) (*roleSchema, error) {
	var out *roleSchema
	j, _ := json.MarshalIndent(v, "", "    ")
	err := json.Unmarshal(j, &out)
	return out, err
}

func memberHasRole(roles []string, roleID string) bool {
	for _, r := range roles {
		if r == roleID {
			return true
		}
	}
	return false
}

func removeRoleID(roles []string, roleID string) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		if r != roleID {
			out = append(out, r)
		}
	}
	return out
}

func resourceDiscordMemberRoles() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMemberRolesCreate,
		ReadContext:   resourceMemberRolesRead,
		UpdateContext: resourceMemberRolesUpdate,
		DeleteContext: resourceMemberRolesDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"user_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"role": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"has_role": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
					},
				},
			},
		},
	}
}

func resourceMemberRolesCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	serverID := d.Get("server_id").(string)
	userID := d.Get("user_id").(string)

	// Validate member exists.
	var member restMemberForRoles
	if err := c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/members/"+userID, nil, nil, &member); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(Hashcode(fmt.Sprintf("%s:%s", serverID, userID))))

	diags := resourceMemberRolesRead(ctx, d, m)
	diags = append(diags, resourceMemberRolesUpdate(ctx, d, m)...)
	return diags
}

func resourceMemberRolesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	userID := d.Get("user_id").(string)

	var member restMemberForRoles
	if err := c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/members/"+userID, nil, nil, &member); err != nil {
		return diag.FromErr(err)
	}

	items := d.Get("role").(*schema.Set).List()
	out := make([]map[string]interface{}, 0, len(items))

	for _, r := range items {
		v, _ := convertToRoleSchema(r)
		out = append(out, map[string]interface{}{
			"role_id":  v.RoleID,
			"has_role": memberHasRole(member.Roles, v.RoleID),
		})
	}

	_ = d.Set("role", out)
	return nil
}

func resourceMemberRolesUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	userID := d.Get("user_id").(string)

	var member restMemberForRoles
	if err := c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/members/"+userID, nil, nil, &member); err != nil {
		return diag.FromErr(err)
	}

	old, newV := d.GetChange("role")
	oldItems := old.(*schema.Set).List()
	items := newV.(*schema.Set).List()

	roles := member.Roles

	for _, r := range items {
		v, _ := convertToRoleSchema(r)
		has := memberHasRole(roles, v.RoleID)
		if v.HasRole && !has {
			roles = append(roles, v.RoleID)
		}
		if !v.HasRole && has {
			roles = removeRoleID(roles, v.RoleID)
		}
	}

	for _, r := range oldItems {
		v, _ := convertToRoleSchema(r)
		if wasRemoved(items, v) && v.HasRole {
			roles = removeRoleID(roles, v.RoleID)
		}
	}

	if err := c.DoJSON(ctx, "PATCH", "/guilds/"+serverID+"/members/"+userID, nil, restMemberRoles{Roles: roles}, nil); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func wasRemoved(items []interface{}, v *roleSchema) bool {
	for _, i := range items {
		item, _ := convertToRoleSchema(i)
		if item.RoleID == v.RoleID {
			return false
		}
	}
	return true
}

func resourceMemberRolesDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	serverID := d.Get("server_id").(string)
	userID := d.Get("user_id").(string)

	var member restMemberForRoles
	if err := c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/members/"+userID, nil, nil, &member); err != nil {
		return diag.FromErr(err)
	}

	items := d.Get("role").(*schema.Set).List()
	roles := member.Roles
	for _, r := range items {
		v, _ := convertToRoleSchema(r)
		has := memberHasRole(roles, v.RoleID)
		if has && v.HasRole {
			roles = removeRoleID(roles, v.RoleID)
		}
	}

	if err := c.DoJSON(ctx, "PATCH", "/guilds/"+serverID+"/members/"+userID, nil, restMemberRoles{Roles: roles}, nil); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
