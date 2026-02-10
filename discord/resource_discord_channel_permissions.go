package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
)

type restPermOverwrite struct {
	ID    string `json:"id"`
	Type  int    `json:"type"` // 0=role, 1=member
	Allow string `json:"allow"`
	Deny  string `json:"deny"`
}

type restChannelOverwrites struct {
	ID                   string              `json:"id"`
	PermissionOverwrites []restPermOverwrite `json:"permission_overwrites"`
}

// discord_channel_permissions manages the full set of permission overwrites on a channel.
// This is an authoritative resource: it will delete any overwrites on the channel that
// are not declared in this resource.
func resourceDiscordChannelPermissions() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDiscordChannelPermissionsUpsert,
		ReadContext:   resourceDiscordChannelPermissionsRead,
		UpdateContext: resourceDiscordChannelPermissionsUpsert,
		DeleteContext: resourceDiscordChannelPermissionsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"channel_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"overwrite": {
				Type:     schema.TypeSet,
				Required: true,
				Set: func(v interface{}) int {
					m := v.(map[string]interface{})
					return Hashcode(fmt.Sprintf("%s:%s", m["type"].(string), m["overwrite_id"].(string)))
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "role or user",
						},
						"overwrite_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"allow": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"allow_bits64": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Allow bitset as 64-bit integer string (decimal or 0x...). Prefer this for newer high-bit permissions.",
						},
						"deny": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"deny_bits64": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Deny bitset as 64-bit integer string (decimal or 0x...). Prefer this for newer high-bit permissions.",
						},
					},
				},
			},
			"reason": {
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool { return true },
			},
		},
	}
}

type owKey struct {
	Type string
	ID   string
}

func owTypeToInt(t string) (int, error) {
	switch t {
	case "role":
		return 0, nil
	case "user":
		return 1, nil
	default:
		return 0, fmt.Errorf("invalid overwrite type %q (expected role or user)", t)
	}
}

func owTypeFromInt(t int) string {
	switch t {
	case 0:
		return "role"
	case 1:
		return "user"
	default:
		return fmt.Sprintf("%d", t)
	}
}

func desiredOverwritesV2(d *schema.ResourceData) (map[owKey]map[string]interface{}, error) {
	out := map[owKey]map[string]interface{}{}
	items := d.Get("overwrite").(*schema.Set).List()
	for _, it := range items {
		m := it.(map[string]interface{})
		typ := m["type"].(string)
		oid := m["overwrite_id"].(string)
		ti, err := owTypeToInt(typ)
		if err != nil {
			return nil, err
		}

		allow64, err := normalizeUint64String(m["allow_bits64"].(string))
		if err != nil {
			return nil, fmt.Errorf("invalid allow_bits64 for overwrite %s: %w", oid, err)
		}
		deny64, err := normalizeUint64String(m["deny_bits64"].(string))
		if err != nil {
			return nil, fmt.Errorf("invalid deny_bits64 for overwrite %s: %w", oid, err)
		}

		allowStr := allow64
		if allowStr == "" {
			if v, ok := m["allow"]; ok && v != nil {
				allowStr = intToUint64String(v.(int))
			} else {
				allowStr = "0"
			}
		}
		denyStr := deny64
		if denyStr == "" {
			if v, ok := m["deny"]; ok && v != nil {
				denyStr = intToUint64String(v.(int))
			} else {
				denyStr = "0"
			}
		}

		out[owKey{Type: typ, ID: oid}] = map[string]interface{}{
			"type":  ti,
			"allow": allowStr,
			"deny":  denyStr,
		}
	}
	return out, nil
}

func readChannelOverwrites(ctx context.Context, c *RestClient, channelID string) (*restChannelOverwrites, error) {
	var out restChannelOverwrites
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/channels/%s", channelID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func resourceDiscordChannelPermissionsUpsert(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	channelID := d.Get("channel_id").(string)
	reason := d.Get("reason").(string)

	ch, err := readChannelOverwrites(ctx, c, channelID)
	if err != nil {
		return diag.FromErr(err)
	}

	want, err := desiredOverwritesV2(d)
	if err != nil {
		return diag.FromErr(err)
	}

	// Delete existing overwrites not declared.
	for _, ow := range ch.PermissionOverwrites {
		k := owKey{Type: owTypeFromInt(ow.Type), ID: ow.ID}
		if _, ok := want[k]; !ok {
			if err := c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/channels/%s/permissions/%s", channelID, ow.ID), nil, nil, nil, reason); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	// Upsert desired overwrites.
	for k, body := range want {
		if err := c.DoJSONWithReason(ctx, "PUT", fmt.Sprintf("/channels/%s/permissions/%s", channelID, k.ID), nil, body, nil, reason); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(channelID)
	return resourceDiscordChannelPermissionsRead(ctx, d, m)
}

func resourceDiscordChannelPermissionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	channelID := d.Get("channel_id").(string)

	ch, err := readChannelOverwrites(ctx, c, channelID)
	if err != nil {
		if IsDiscordHTTPStatus(err, 404) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	outs := make([]map[string]interface{}, 0, len(ch.PermissionOverwrites))
	for _, ow := range ch.PermissionOverwrites {
		allowNorm, err := normalizeUint64String(ow.Allow)
		if err != nil && ow.Allow != "" {
			return diag.Errorf("failed to parse overwrite allow bits for %s: %s", ow.ID, err.Error())
		}
		denyNorm, err := normalizeUint64String(ow.Deny)
		if err != nil && ow.Deny != "" {
			return diag.Errorf("failed to parse overwrite deny bits for %s: %s", ow.ID, err.Error())
		}
		allowInt := 0
		if allowNorm != "" {
			av, _ := strconv.ParseUint(allowNorm, 10, 64)
			if i, ierr := uint64ToIntIfFits(av); ierr == nil {
				allowInt = i
			}
		}
		denyInt := 0
		if denyNorm != "" {
			dv, _ := strconv.ParseUint(denyNorm, 10, 64)
			if i, ierr := uint64ToIntIfFits(dv); ierr == nil {
				denyInt = i
			}
		}
		outs = append(outs, map[string]interface{}{
			"type":         owTypeFromInt(ow.Type),
			"overwrite_id": ow.ID,
			"allow":        allowInt,
			"allow_bits64": allowNorm,
			"deny":         denyInt,
			"deny_bits64":  denyNorm,
		})
	}

	d.SetId(channelID)
	_ = d.Set("overwrite", outs)
	return nil
}

func resourceDiscordChannelPermissionsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Context).Rest
	channelID := d.Get("channel_id").(string)
	reason := d.Get("reason").(string)

	// Remove all overwrites declared in this resource.
	items := d.Get("overwrite").(*schema.Set).List()
	for _, it := range items {
		mm := it.(map[string]interface{})
		oid := mm["overwrite_id"].(string)
		if err := c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/channels/%s/permissions/%s", channelID, oid), nil, nil, nil, reason); err != nil {
			if IsDiscordHTTPStatus(err, 404) {
				continue
			}
			return diag.FromErr(err)
		}
	}
	return nil
}
