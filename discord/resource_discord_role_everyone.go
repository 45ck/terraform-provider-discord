package discord

import (
	"github.com/andersfylling/disgord"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/context"
	"strconv"
	"strings"
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
				ForceNew: false,
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
	data.SetId(data.Id())
	data.Set("server_id", getId(data.Id()).String())

	return schema.ImportStatePassthroughContext(ctx, data, i)
}

func resourceRoleEveryoneRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	serverId := getId(d.Get("server_id").(string))
	d.SetId(serverId.String())

	server, err := client.GetGuild(ctx, serverId)
	if err != nil {
		return diag.Errorf("Failed to fetch server %s: %s", serverId.String(), err.Error())
	}

	role, err := server.Role(serverId)
	if err != nil {
		return diag.Errorf("Failed to fetch role %s: %s", d.Id(), err.Error())
	}

	d.Set("permissions", role.Permissions)
	_ = d.Set("permissions_bits64", strconv.FormatUint(uint64(role.Permissions), 10))

	return diags
}

func resourceRoleEveryoneUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	serverId := getId(d.Get("server_id").(string))
	d.SetId(serverId.String())
	builder := client.UpdateGuildRole(ctx, serverId, serverId)

	perms := uint64(d.Get("permissions").(int))
	if s := strings.TrimSpace(d.Get("permissions_bits64").(string)); s != "" {
		v, err := uint64StringToPermissionBit(s)
		if err != nil {
			return diag.Errorf("invalid permissions_bits64: %s", err.Error())
		}
		perms = v
	}
	builder.SetPermissions(disgord.PermissionBit(perms))

	role, err := builder.Execute()
	if err != nil {
		return diag.Errorf("Failed to update role %s: %s", d.Id(), err.Error())
	}

	d.Set("permissions", role.Permissions)
	_ = d.Set("permissions_bits64", strconv.FormatUint(uint64(role.Permissions), 10))

	return diags
}
