# Discord Terraform Provider

This is a fork of [aequasi/terraform-provider-discord](https://github.com/aequasi/terraform-provider-discord). We ran into some problems with this provider and decided to fix them with this opinionated version.

https://registry.terraform.io/providers/Chaotic-Logic/discord/latest

Note: this fork serves the provider over Terraform plugin protocol v6, which requires Terraform CLI 1.0+.

## Examples

See:

* `examples/full_server` (create roles/channels, set permissions, pin a rules message)
* `examples/guild_settings` (escape hatch for guild/server settings via `discord_guild_settings`)
* `examples/api_resource_widget` (generic JSON REST escape hatch via `discord_api_resource`)

## Building the provider
### Development
```sh
go mod vendor
make
```

### Release
```
go mod vendor
export GPG_FINGERPRINT="D081560F57E59EDA7CB369BE2FFBD6BE37B85C17"
goreleaser release --skip-publish
```

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
* discord_system_channel
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
