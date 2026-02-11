# Discord Webhook Resource

Manages a channel webhook.

## Example Usage

```hcl-terraform
resource "discord_webhook" "alerts" {
  channel_id = discord_text_channel.alerts.id
  name       = "alerts"
}
```

## Argument Reference

* `channel_id` (Required) Channel ID
* `name` (Required) Webhook name
* `avatar_data_uri` (Optional) Webhook avatar as data URI

## Attribute Reference

* `token` Webhook token (sensitive)
* `url` Webhook URL
* `guild_id` Guild ID

