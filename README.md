# Discord Terraform Provider

This is a fork of [aequasi/terraform-provider-discord](https://github.com/aequasi/terraform-provider-discord). We ran into some problems with this provider and decided to fix them with this opinionated version.

https://registry.terraform.io/providers/Chaotic-Logic/discord/latest

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

* discord_category_channel
* discord_channel
* discord_channel_order
* discord_channel_permission
* discord_channel_permissions
* discord_invite
* discord_guild_settings
* discord_member_roles
* discord_member_nickname
* discord_member_timeout
* discord_ban
* discord_message
* discord_thread
* discord_thread_member
* discord_role
* discord_role_everyone
* discord_role_order
* discord_server
* discord_system_channel
* discord_text_channel
* discord_voice_channel
* discord_welcome_screen
* discord_onboarding
* discord_member_verification
* discord_automod_rule
* discord_scheduled_event
* discord_emoji
* discord_sticker
* discord_webhook
* discord_stage_instance
* discord_api_resource
* discord_soundboard_sound

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
