# Discord Widget Settings Resource

Manages a guild's server widget settings (GET/PATCH `/guilds/{guild.id}/widget`).

Note: destroying this resource removes it from Terraform state only; it does not revert widget settings on Discord.

## Example Usage

```hcl-terraform
resource "discord_widget_settings" "this" {
  server_id = var.server_id
  enabled   = true
  channel_id = discord_channel.rules.id

  reason = "Managed by Terraform"
}
```

## Argument Reference

* `server_id` (Required) Guild (server) ID.
* `enabled` (Required) Whether the widget is enabled.
* `channel_id` (Optional) Widget channel ID. Required when `enabled = true`.
* `reason` (Optional) Audit log reason (not read back).

## Attribute Reference

* `id` Internal Terraform ID (equal to `server_id`).

