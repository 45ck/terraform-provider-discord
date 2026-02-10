package discord

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"math"
	"strconv"
	"strings"
)

var permissions map[string]uint64

func dataSourceDiscordPermission() *schema.Resource {
	// Note: this data source intentionally exposes a wide set of permissions, including newer
	// high-bit flags (e.g. polls, external apps). This provider is effectively 64-bit only when
	// using these higher bits because Terraform SDK uses Go int for TypeInt.
	permissions = map[string]uint64{
		// Classic permissions.
		"create_instant_invite": 1 << 0,
		"kick_members":          1 << 1,
		"ban_members":           1 << 2,
		"administrator":         1 << 3,
		"manage_channels":       1 << 4,
		"manage_guild":          1 << 5,
		"add_reactions":         1 << 6,
		"view_audit_log":        1 << 7,
		"priority_speaker":      1 << 8,
		"stream":                1 << 9,
		"view_channel":          1 << 10,
		"send_messages":         1 << 11,
		"send_tts_messages":     1 << 12,
		"manage_messages":       1 << 13,
		"embed_links":           1 << 14,
		"attach_files":          1 << 15,
		"read_message_history":  1 << 16,
		"mention_everyone":      1 << 17,
		"use_external_emojis":   1 << 18,
		"view_guild_insights":   1 << 19,
		"connect":               1 << 20,
		"speak":                 1 << 21,
		"mute_members":          1 << 22,
		"deafen_members":        1 << 23,
		"move_members":          1 << 24,
		"use_vad":               1 << 25,
		"change_nickname":       1 << 26,
		"manage_nicknames":      1 << 27,
		"manage_roles":          1 << 28,
		"manage_webhooks":       1 << 29,

		// Expressions.
		"manage_expressions":       1 << 30,
		"manage_guild_expressions": 1 << 30, // alias
		"manage_emojis":            1 << 30, // backwards-compatible alias

		// Modern permissions.
		"use_application_commands":            1 << 31,
		"request_to_speak":                    1 << 32,
		"manage_events":                       1 << 33,
		"manage_threads":                      1 << 34,
		"create_public_threads":               1 << 35,
		"create_private_threads":              1 << 36,
		"use_external_stickers":               1 << 37,
		"send_messages_in_threads":            1 << 38,
		"use_embedded_activities":             1 << 39,
		"start_embedded_activities":           1 << 39, // alias
		"moderate_members":                    1 << 40,
		"view_creator_monetization_analytics": 1 << 41,
		"use_soundboard":                      1 << 42,
		"create_expressions":                  1 << 43,
		"create_events":                       1 << 44,
		"use_external_sounds":                 1 << 45,
		"send_voice_messages":                 1 << 46,
		"use_clyde_ai":                        1 << 47, // deprecated/removed in many clients, but retained for completeness
		"set_voice_channel_status":            1 << 48,
		"send_polls":                          1 << 49,
		"use_external_apps":                   1 << 50,
		"pin_messages":                        1 << 51,
		"bypass_slowmode":                     1 << 52,
	}

	schemaMap := make(map[string]*schema.Schema)
	schemaMap["allow_extends"] = &schema.Schema{
		Type:     schema.TypeInt,
		Optional: true,
	}
	schemaMap["allow_extends_bits64"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Additional allow permission bits as a 64-bit integer string (decimal or 0x...).",
	}
	schemaMap["deny_extends"] = &schema.Schema{
		Type:     schema.TypeInt,
		Optional: true,
	}
	schemaMap["deny_extends_bits64"] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Additional deny permission bits as a 64-bit integer string (decimal or 0x...).",
	}
	schemaMap["allow_bits"] = &schema.Schema{
		Type:     schema.TypeInt,
		Computed: true,
	}
	schemaMap["allow_bits64"] = &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Allow permission bits as a 64-bit integer string (decimal).",
	}
	schemaMap["deny_bits"] = &schema.Schema{
		Type:     schema.TypeInt,
		Computed: true,
	}
	schemaMap["deny_bits64"] = &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Deny permission bits as a 64-bit integer string (decimal).",
	}
	for k := range permissions {
		schemaMap[k] = &schema.Schema{
			Optional: true,
			Type:     schema.TypeString,
			Default:  "unset",
			ValidateDiagFunc: func(v interface{}, path cty.Path) (diags diag.Diagnostics) {
				str := v.(string)
				allowed := [3]string{"allow", "unset", "deny"}

				if !contains(allowed, str) {
					return append(diags, diag.Errorf("%s is not an allowed value. Pick one of: allow, unset, deny", str)...)
				}

				return diags
			},
		}
	}

	return &schema.Resource{
		ReadContext: dataSourceDiscordPermissionRead,
		Schema:      schemaMap,
	}
}

func dataSourceDiscordPermissionRead(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	var allowBits uint64
	var denyBits uint64
	for perm, bit := range permissions {
		v := d.Get(perm).(string)
		if v == "allow" {
			allowBits |= bit
		}
		if v == "deny" {
			denyBits |= bit
		}
	}

	// Apply extends (legacy int and 64-bit string variants).
	allowBits |= uint64(d.Get("allow_extends").(int))
	denyBits |= uint64(d.Get("deny_extends").(int))

	if s := strings.TrimSpace(d.Get("allow_extends_bits64").(string)); s != "" {
		v, err := strconv.ParseUint(s, 0, 64)
		if err != nil {
			return diag.Errorf("allow_extends_bits64 must be a 64-bit integer string: %s", err.Error())
		}
		allowBits |= v
	}
	if s := strings.TrimSpace(d.Get("deny_extends_bits64").(string)); s != "" {
		v, err := strconv.ParseUint(s, 0, 64)
		if err != nil {
			return diag.Errorf("deny_extends_bits64 must be a 64-bit integer string: %s", err.Error())
		}
		denyBits |= v
	}

	d.SetId(strconv.Itoa(Hashcode(fmt.Sprintf("%d:%d", allowBits, denyBits))))

	// Always expose 64-bit-safe strings.
	_ = d.Set("allow_bits64", strconv.FormatUint(allowBits, 10))
	_ = d.Set("deny_bits64", strconv.FormatUint(denyBits, 10))

	// Best-effort legacy ints. On 32-bit these can overflow for newer high-bit permissions.
	maxInt := uint64(math.MaxInt)
	if allowBits <= maxInt {
		_ = d.Set("allow_bits", int(allowBits))
	} else {
		_ = d.Set("allow_bits", 0)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "allow_bits overflowed int; use allow_bits64 instead",
			Detail:   fmt.Sprintf("allow_bits=%d does not fit in TypeInt on this platform", allowBits),
		})
	}
	if denyBits <= maxInt {
		_ = d.Set("deny_bits", int(denyBits))
	} else {
		_ = d.Set("deny_bits", 0)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "deny_bits overflowed int; use deny_bits64 instead",
			Detail:   fmt.Sprintf("deny_bits=%d does not fit in TypeInt on this platform", denyBits),
		})
	}

	return diags
}
