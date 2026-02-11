# Discord Stickers Data Source

Lists stickers in a guild.

## Example Usage

```hcl-terraform
data "discord_stickers" "all" {
  server_id = var.server_id
}
```

## Argument Reference

* `server_id` (Required) Server ID

## Attribute Reference

* `sticker` List:
  * `id`
  * `name`
  * `description`
  * `tags`
  * `format_type`

