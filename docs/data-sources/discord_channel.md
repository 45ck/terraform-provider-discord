# Discord Channel Data Source

Looks up a channel by name within a server.

## Example Usage

```hcl-terraform
data "discord_channel" "rules" {
  server_id = var.server_id
  name      = "rules"
  type      = "text"
}

output rules_channel_id {
  value = data.discord_channel.rules.id
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `name` (Required) Channel name
* `type` (Optional) Channel type filter

## Attribute Reference

* `id` Channel ID
* `parent_id` Parent category ID (if any)

