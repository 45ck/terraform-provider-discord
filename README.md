# Discord Terraform Provider

This is a fork of [aequasi/terraform-provider-discord](https://github.com/aequasi/terraform-provider-discord). We ran into some problems with this provider and decided to fix them with this opinionated version.

https://registry.terraform.io/providers/Chaotic-Logic/discord/latest

Note: this fork serves the provider over Terraform plugin protocol v6, which requires Terraform CLI 1.0+.

## Project Status

This repository is a maintained community fork.

Lineage:

* Upstream (original): `aequasi/terraform-provider-discord`
* Prior maintained fork: `Chaotic-Logic/terraform-provider-discord`
* This fork: `45ck/terraform-provider-discord`

This project is not affiliated with Discord.

## Module Path

The Go module path for this repo is `github.com/45ck/terraform-provider-discord`.

This does not affect Terraform provider installation (which uses the provider source address in your Terraform configuration).

## License

GPL-3.0 (see `LICENSE`).

## Examples

See:

* `examples/full_server` (create roles/channels, set permissions, pin a rules message)
* `examples/admin_no_clickops` (manage server basics, threads, webhooks, optional stickers/soundboard)
* `examples/guild_settings` (escape hatch for guild/server settings via `discord_guild_settings`)
* `examples/api_resource_widget` (generic JSON REST escape hatch via `discord_api_resource`)

## Building the provider
### Development
```sh
go mod vendor
go test ./...
go build ./...
```

Note: `make` targets are primarily intended for *nix environments. On Windows, prefer the direct `go ...` commands above.

### Release
```
go mod vendor
export GPG_FINGERPRINT="D081560F57E59EDA7CB369BE2FFBD6BE37B85C17"
goreleaser release --skip-publish
```

Acceptance tests are opt-in. See `docs/ACCEPTANCE_TESTS.md` (and `scripts/testacc.ps1` for a PowerShell helper).

## Security

Please report vulnerabilities via GitHub Security Advisories. See `SECURITY.md`.

## Resources

First-class resources (Framework):

* discord_api_resource (generic CRUD "escape hatch")
* discord_automod_rule (JSON passthrough)
* discord_ban
* discord_channel (supports `type = "category" | "text" | "voice" | ...`)
* discord_channel_order (bulk ordering/moves)
* discord_channel_permission (single overwrite)
* discord_channel_permissions (authoritative permission overwrites)
* discord_emoji
* discord_guild_template
* discord_guild_template_sync (sync action resource; bump `sync_nonce` to force resync)
* discord_guild_settings (generic guild PATCH "escape hatch")
* discord_invite
* discord_member_nickname
* discord_member_roles
* discord_member_timeout
* discord_member_verification (JSON passthrough)
* discord_message
* discord_onboarding (JSON passthrough)
* discord_role
* discord_role_everyone
* discord_role_order (bulk ordering)
* discord_scheduled_event
* discord_server
* discord_soundboard_sound
* discord_stage_instance
* discord_sticker
* discord_system_channel
* discord_thread
* discord_thread_member
* discord_widget_settings
* discord_welcome_screen
* discord_webhook

If you need an endpoint that does not have a first-class resource yet, use `discord_api_resource` or `discord_guild_settings` to eliminate "clickops".

## Data

* discord_color
* discord_local_image
* discord_permission
* discord_role
* discord_server
* discord_member
* discord_system_channel
* discord_channel
* discord_api_request
* discord_thread_members
* discord_emojis
* discord_stickers
* discord_soundboard_sounds
* discord_soundboard_default_sounds

## Todo

#### Data Sources

Legacy per-type channel data sources (`discord_text_channel`, `discord_voice_channel`, etc.) are not planned.

Use `discord_channel` (by name lookup) or `discord_api_request` for generic GET access instead.

## Import IDs

Some Discord objects are only addressable via a guild-scoped route. For those resources, import requires a composite ID:

* `discord_automod_rule`: `server_id:rule_id`
* `discord_emoji`: `server_id:emoji_id`
* `discord_role`: `server_id:role_id`
* `discord_scheduled_event`: `server_id:event_id`
* `discord_member_roles`: `server_id:user_id`
* `discord_member_nickname`: `server_id:user_id`
* `discord_member_timeout`: `server_id:user_id`
* `discord_message`: `channel_id:message_id`
* `discord_channel_permission`: `channel_id:overwrite_id:type`
* `discord_sticker`: `server_id:sticker_id`
* `discord_soundboard_sound`: `server_id:sound_id`
* `discord_thread_member`: `thread_id:user_id`
* `discord_guild_template`: `server_id:template_code`
* `discord_guild_template_sync`: `server_id:template_code`
* `discord_widget_settings`: `server_id`
