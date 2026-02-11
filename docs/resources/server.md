# Discord Server Resource

A resource to manage a Discord server (guild).

Important: Discord bot tokens generally cannot create new guilds via the public API.
In practice, you should create the server out-of-band and then `terraform import` it
into state to manage it.

Important: Destroying `discord_server` will **not** delete the guild. This is a safety
measure to prevent accidental server deletion.

## Example Usage

```hcl-terraform
resource "discord_server" "my_server" {
  # Adopt an existing guild.
  server_id = var.server_id
  name      = "My Awesome Server"

  # Typical admin defaults. Adjust as needed.
  default_message_notifications = 1
  verification_level           = 1
  explicit_content_filter      = 2
  afk_timeout                  = 300

  # Write-only upload (Discord only returns hashes).
  # Use an empty string to clear.
  icon_data_uri = data.discord_local_image.logo.data_uri

  # Optional audit log reason (not read back).
  reason = "Managed by Terraform"
}
```

## Argument Reference

* `server_id` (Required) Guild (server) ID to manage (adopt).
* `name` (Required) Name of the server.
* `default_message_notifications` (Optional) Default Message Notification settings (0 = all messages, 1 = mentions).
* `verification_level` (Optional) Verification Level of the server.
* `explicit_content_filter` (Optional) Explicit Content Filter level.
* `afk_channel_id` (Optional) Channel ID for moving AFK users to. Use an empty string to clear.
* `afk_timeout` (Optional) How many seconds before moving an AFK user.
* `icon_data_uri` (Optional) Data URI of an image to set the icon. Use an empty string to clear.
* `splash_data_uri` (Optional) Data URI of an image to set the splash. Use an empty string to clear.
* `owner_id` (Optional) Owner ID of the server (transfers ownership). This is privileged and often not permitted for bot tokens.
* `reason` (Optional) Audit log reason (not read back).

Note: system messages are managed via the separate `discord_system_channel` resource.
For settings not covered by this schema, use `discord_guild_settings` or the generic
`discord_api_resource` escape hatch.

## Attribute Reference

* `id` Internal Terraform ID (equal to `server_id`).
* `icon_hash` Hash of the icon
* `splash_hash` Hash of the splash
