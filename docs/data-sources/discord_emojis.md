# Discord Emojis Data Source

Lists emojis in a guild.

## Example Usage

```hcl-terraform
data discord_emojis all {
  server_id = var.server_id
}
```

## Argument Reference

* `server_id` (Required) Server ID

## Attribute Reference

* `emoji` List:
  * `id`
  * `name`
  * `managed`
  * `animated`

