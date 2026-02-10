package fw

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewPermissionDataSource() datasource.DataSource {
	return &permissionDataSource{}
}

type permissionDataSource struct{}

// Keep this list aligned with discord/data_source_discord_permission.go.
var permissionBits = map[string]uint64{
	"create_instant_invite":    1 << 0,
	"kick_members":             1 << 1,
	"ban_members":              1 << 2,
	"administrator":            1 << 3,
	"manage_channels":          1 << 4,
	"manage_guild":             1 << 5,
	"add_reactions":            1 << 6,
	"view_audit_log":           1 << 7,
	"priority_speaker":         1 << 8,
	"stream":                   1 << 9,
	"view_channel":             1 << 10,
	"send_messages":            1 << 11,
	"send_tts_messages":        1 << 12,
	"manage_messages":          1 << 13,
	"embed_links":              1 << 14,
	"attach_files":             1 << 15,
	"read_message_history":     1 << 16,
	"mention_everyone":         1 << 17,
	"use_external_emojis":      1 << 18,
	"view_guild_insights":      1 << 19,
	"connect":                  1 << 20,
	"speak":                    1 << 21,
	"mute_members":             1 << 22,
	"deafen_members":           1 << 23,
	"move_members":             1 << 24,
	"use_vad":                  1 << 25,
	"change_nickname":          1 << 26,
	"manage_nicknames":         1 << 27,
	"manage_roles":             1 << 28,
	"manage_webhooks":          1 << 29,
	"manage_expressions":       1 << 30,
	"manage_guild_expressions": 1 << 30,
	"manage_emojis":            1 << 30,

	"use_application_commands":            1 << 31,
	"request_to_speak":                    1 << 32,
	"manage_events":                       1 << 33,
	"manage_threads":                      1 << 34,
	"create_public_threads":               1 << 35,
	"create_private_threads":              1 << 36,
	"use_external_stickers":               1 << 37,
	"send_messages_in_threads":            1 << 38,
	"use_embedded_activities":             1 << 39,
	"start_embedded_activities":           1 << 39,
	"moderate_members":                    1 << 40,
	"view_creator_monetization_analytics": 1 << 41,
	"use_soundboard":                      1 << 42,
	"create_expressions":                  1 << 43,
	"create_events":                       1 << 44,
	"use_external_sounds":                 1 << 45,
	"send_voice_messages":                 1 << 46,
	"use_clyde_ai":                        1 << 47,
	"set_voice_channel_status":            1 << 48,
	"send_polls":                          1 << 49,
	"use_external_apps":                   1 << 50,
	"pin_messages":                        1 << 51,
	"bypass_slowmode":                     1 << 52,
}

func (d *permissionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission"
}

func (d *permissionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed: true,
		},
		"allow_extends": schema.Int64Attribute{
			Optional: true,
		},
		"allow_extends_bits64": schema.StringAttribute{
			Optional:    true,
			Description: "Additional allow permission bits as a 64-bit integer string (decimal or 0x...).",
		},
		"deny_extends": schema.Int64Attribute{
			Optional: true,
		},
		"deny_extends_bits64": schema.StringAttribute{
			Optional:    true,
			Description: "Additional deny permission bits as a 64-bit integer string (decimal or 0x...).",
		},
		"allow_bits": schema.Int64Attribute{
			Computed: true,
		},
		"allow_bits64": schema.StringAttribute{
			Computed:    true,
			Description: "Allow permission bits as a 64-bit integer string (decimal).",
		},
		"deny_bits": schema.Int64Attribute{
			Computed: true,
		},
		"deny_bits64": schema.StringAttribute{
			Computed:    true,
			Description: "Deny permission bits as a 64-bit integer string (decimal).",
		},
	}

	for k := range permissionBits {
		attrs[k] = schema.StringAttribute{
			Optional: true,
			// Default to "unset" by convention.
			Computed: true,
		}
	}

	resp.Schema = schema.Schema{
		Attributes: attrs,
	}
}

func (d *permissionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var allowExtends types.Int64
	var denyExtends types.Int64
	var allowExtends64 types.String
	var denyExtends64 types.String

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("allow_extends"), &allowExtends)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("deny_extends"), &denyExtends)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("allow_extends_bits64"), &allowExtends64)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("deny_extends_bits64"), &denyExtends64)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var allowBits uint64
	var denyBits uint64
	for perm, bit := range permissionBits {
		var v types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(perm), &v)...)
		if resp.Diagnostics.HasError() {
			return
		}
		vs := strings.TrimSpace(v.ValueString())
		if vs == "" {
			vs = "unset"
		}

		switch vs {
		case "allow":
			allowBits |= bit
		case "deny":
			denyBits |= bit
		case "unset":
		default:
			resp.Diagnostics.AddError("Invalid permission value", fmt.Sprintf("%s must be one of allow, unset, deny", perm))
			return
		}

		// Preserve the flag value in state for consistency.
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(perm), types.StringValue(vs))...)
	}

	if !allowExtends.IsNull() && !allowExtends.IsUnknown() {
		allowBits |= uint64(allowExtends.ValueInt64())
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("allow_extends"), allowExtends)...)
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("allow_extends"), types.Int64Value(0))...)
	}
	if !denyExtends.IsNull() && !denyExtends.IsUnknown() {
		denyBits |= uint64(denyExtends.ValueInt64())
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deny_extends"), denyExtends)...)
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deny_extends"), types.Int64Value(0))...)
	}

	if s := strings.TrimSpace(allowExtends64.ValueString()); s != "" {
		v, err := strconv.ParseUint(s, 0, 64)
		if err != nil {
			resp.Diagnostics.AddError("Invalid allow_extends_bits64", err.Error())
			return
		}
		allowBits |= v
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("allow_extends_bits64"), types.StringValue(s))...)
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("allow_extends_bits64"), types.StringValue(""))...)
	}
	if s := strings.TrimSpace(denyExtends64.ValueString()); s != "" {
		v, err := strconv.ParseUint(s, 0, 64)
		if err != nil {
			resp.Diagnostics.AddError("Invalid deny_extends_bits64", err.Error())
			return
		}
		denyBits |= v
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deny_extends_bits64"), types.StringValue(s))...)
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deny_extends_bits64"), types.StringValue(""))...)
	}

	id := strconv.Itoa(discord.Hashcode(fmt.Sprintf("%d:%d", allowBits, denyBits)))
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(id))...)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("allow_bits64"), types.StringValue(strconv.FormatUint(allowBits, 10)))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deny_bits64"), types.StringValue(strconv.FormatUint(denyBits, 10)))...)

	// These are stable 64-bit numbers in framework.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("allow_bits"), types.Int64Value(int64(allowBits)))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deny_bits"), types.Int64Value(int64(denyBits)))...)
}
