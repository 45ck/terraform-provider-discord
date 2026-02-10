package discord

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type restPermOverwriteRead struct {
	ID    string `json:"id"`
	Type  int    `json:"type"` // 0=role, 1=member
	Allow string `json:"allow"`
	Deny  string `json:"deny"`
}

type restChannelPermsRead struct {
	ID                   string                  `json:"id"`
	PermissionOverwrites []restPermOverwriteRead `json:"permission_overwrites"`
}

type restPermOverwriteUpsert struct {
	Allow string `json:"allow"`
	Deny  string `json:"deny"`
	Type  int    `json:"type"`
}

func resourceDiscordChannelPermission() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceChannelPermissionUpsert,
		ReadContext:   resourceChannelPermissionRead,
		UpdateContext: resourceChannelPermissionUpsert,
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

func owTypeToIntLegacy(t string) (int, error) {
	switch t {
	case "role":
		return 0, nil
	case "user":
		return 1, nil
	default:
		return 0, fmt.Errorf("invalid overwrite type %q (expected role or user)", t)
	}
}

func resourceChannelPermissionUpsert(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	channelID := d.Get("channel_id").(string)
	overwriteID := d.Get("overwrite_id").(string)
	typ, err := owTypeToIntLegacy(d.Get("type").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	allow := uint64(d.Get("allow").(int))
	if s := strings.TrimSpace(d.Get("allow_bits64").(string)); s != "" {
		v, err := uint64StringToPermissionBit(s)
		if err != nil {
			return diag.Errorf("invalid allow_bits64: %s", err.Error())
		}
		allow = v
	}
	deny := uint64(d.Get("deny").(int))
	if s := strings.TrimSpace(d.Get("deny_bits64").(string)); s != "" {
		v, err := uint64StringToPermissionBit(s)
		if err != nil {
			return diag.Errorf("invalid deny_bits64: %s", err.Error())
		}
		deny = v
	}

	body := restPermOverwriteUpsert{
		Allow: strconv.FormatUint(allow, 10),
		Deny:  strconv.FormatUint(deny, 10),
		Type:  typ,
	}

	if err := c.DoJSON(ctx, "PUT", fmt.Sprintf("/channels/%s/permissions/%s", channelID, overwriteID), nil, body, nil); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(Hashcode(fmt.Sprintf("%s:%s:%s", channelID, overwriteID, d.Get("type").(string)))))
	return resourceChannelPermissionRead(ctx, d, m)
}

func resourceChannelPermissionRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	channelID := d.Get("channel_id").(string)
	overwriteID := d.Get("overwrite_id").(string)
	typ, err := owTypeToIntLegacy(d.Get("type").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	var ch restChannelPermsRead
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/channels/%s", channelID), nil, nil, &ch); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	found := false
	for _, x := range ch.PermissionOverwrites {
		if x.Type == typ && x.ID == overwriteID {
			found = true
			_ = d.Set("allow_bits64", strings.TrimSpace(x.Allow))
			_ = d.Set("deny_bits64", strings.TrimSpace(x.Deny))

			if v, err := uint64StringToPermissionBit(x.Allow); err == nil {
				if i, err := uint64ToIntIfFits(v); err == nil {
					_ = d.Set("allow", i)
				} else {
					_ = d.Set("allow", 0)
				}
			}
			if v, err := uint64StringToPermissionBit(x.Deny); err == nil {
				if i, err := uint64ToIntIfFits(v); err == nil {
					_ = d.Set("deny", i)
				} else {
					_ = d.Set("deny", 0)
				}
			}
			break
		}
	}

	if !found {
		// Treat as gone.
		d.SetId("")
		return nil
	}
	return nil
}

func resourceChannelPermissionDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest

	channelID := d.Get("channel_id").(string)
	overwriteID := d.Get("overwrite_id").(string)

	if err := c.DoJSON(ctx, "DELETE", fmt.Sprintf("/channels/%s/permissions/%s", channelID, overwriteID), nil, nil, nil); err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}
