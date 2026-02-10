package discord

import (
	"fmt"
	"github.com/andersfylling/disgord"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/context"
	"strconv"
	"strings"
)

func resourceDiscordChannelPermission() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceChannelPermissionCreate,
		ReadContext:   resourceChannelPermissionRead,
		UpdateContext: resourceChannelPermissionUpdate,
		DeleteContext: resourceChannelPermissionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"channel_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
				ValidateDiagFunc: func(val interface{}, path cty.Path) (diags diag.Diagnostics) {
					v := val.(string)

					if v != "role" && v != "user" {
						diags = append(diags, diag.Errorf("%s is not a valid type. Must be \"role\" or \"user\"", v)...)
					}

					return diags
				},
			},
			"overwrite_id": {
				ForceNew: true,
				Required: true,
				Type:     schema.TypeString,
			},
			"allow": {
				AtLeastOneOf: []string{"allow", "deny", "allow_bits64", "deny_bits64"},
				Optional:     true,
				Type:         schema.TypeInt,
			},
			"allow_bits64": {
				AtLeastOneOf: []string{"allow", "deny", "allow_bits64", "deny_bits64"},
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "Allow bitset as 64-bit integer string (decimal or 0x...). Prefer this for newer high-bit permissions.",
			},
			"deny": {
				AtLeastOneOf: []string{"allow", "deny", "allow_bits64", "deny_bits64"},
				Optional:     true,
				Type:         schema.TypeInt,
			},
			"deny_bits64": {
				AtLeastOneOf: []string{"allow", "deny", "allow_bits64", "deny_bits64"},
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "Deny bitset as 64-bit integer string (decimal or 0x...). Prefer this for newer high-bit permissions.",
			},
		},
	}
}

func resourceChannelPermissionCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	channelId := getId(d.Get("channel_id").(string))
	overwriteId := getId(d.Get("overwrite_id").(string))

	allow := uint64(d.Get("allow").(int))
	if s := d.Get("allow_bits64").(string); strings.TrimSpace(s) != "" {
		v, err := uint64StringToPermissionBit(s)
		if err != nil {
			return diag.Errorf("invalid allow_bits64: %s", err.Error())
		}
		allow = v
	}
	deny := uint64(d.Get("deny").(int))
	if s := d.Get("deny_bits64").(string); strings.TrimSpace(s) != "" {
		v, err := uint64StringToPermissionBit(s)
		if err != nil {
			return diag.Errorf("invalid deny_bits64: %s", err.Error())
		}
		deny = v
	}

	err := client.UpdateChannelPermissions(ctx, channelId, overwriteId, &disgord.UpdateChannelPermissionsParams{
		Allow: disgord.PermissionBit(allow),
		Deny:  disgord.PermissionBit(deny),
		Type:  d.Get("type").(string),
	})

	if err != nil {
		return diag.Errorf("Failed to update channel permissions %s: %s", channelId.String(), err.Error())
	}

	d.SetId(strconv.Itoa(Hashcode(fmt.Sprintf("%s:%s:%s", channelId, overwriteId, d.Get("type").(string)))))

	return diags
}

func resourceChannelPermissionRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	channelId := getId(d.Get("channel_id").(string))
	overwriteId := getId(d.Get("overwrite_id").(string))

	channel, err := client.GetChannel(ctx, channelId)
	if err != nil {
		return diag.Errorf("Failed to find channel %s: %s", channelId.String(), err.Error())
	}

	for _, x := range channel.PermissionOverwrites {
		if x.Type == d.Get("type").(string) && x.ID == overwriteId {
			_ = d.Set("allow_bits64", strconv.FormatUint(uint64(x.Allow), 10))
			_ = d.Set("deny_bits64", strconv.FormatUint(uint64(x.Deny), 10))

			if i, err := uint64ToIntIfFits(uint64(x.Allow)); err == nil {
				_ = d.Set("allow", i)
			} else {
				_ = d.Set("allow", 0)
			}
			if i, err := uint64ToIntIfFits(uint64(x.Deny)); err == nil {
				_ = d.Set("deny", i)
			} else {
				_ = d.Set("deny", 0)
			}
			break
		}
	}

	return diags
}

func resourceChannelPermissionUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	channelId := getId(d.Get("channel_id").(string))
	overwriteId := getId(d.Get("overwrite_id").(string))

	allow := uint64(d.Get("allow").(int))
	if s := d.Get("allow_bits64").(string); strings.TrimSpace(s) != "" {
		v, err := uint64StringToPermissionBit(s)
		if err != nil {
			return diag.Errorf("invalid allow_bits64: %s", err.Error())
		}
		allow = v
	}
	deny := uint64(d.Get("deny").(int))
	if s := d.Get("deny_bits64").(string); strings.TrimSpace(s) != "" {
		v, err := uint64StringToPermissionBit(s)
		if err != nil {
			return diag.Errorf("invalid deny_bits64: %s", err.Error())
		}
		deny = v
	}

	err := client.UpdateChannelPermissions(ctx, channelId, overwriteId, &disgord.UpdateChannelPermissionsParams{
		Allow: disgord.PermissionBit(allow),
		Deny:  disgord.PermissionBit(deny),
		Type:  d.Get("type").(string),
	})

	if err != nil {
		return diag.Errorf("Failed to update channel permissions %s: %s", channelId.String(), err.Error())
	}

	return diags
}

func resourceChannelPermissionDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	channelId := getId(d.Get("channel_id").(string))
	overwriteId := getId(d.Get("overwrite_id").(string))
	err := client.DeleteChannelPermission(ctx, channelId, overwriteId)

	if err != nil {
		return diag.Errorf("Failed to delete channel permissions %s: %s", channelId.String(), err.Error())
	}

	return diags
}
