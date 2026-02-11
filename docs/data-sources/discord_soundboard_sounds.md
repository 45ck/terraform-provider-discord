# Discord Soundboard Sounds Data Source

Lists soundboard sounds in a guild.

## Example Usage

```hcl-terraform
data "discord_soundboard_sounds" "all" {
  server_id = var.server_id
}
```

## Argument Reference

* `server_id` (Required) Server ID

## Attribute Reference

* `sound` List:
  * `sound_id`
  * `name`
  * `volume`
  * `emoji_id`
  * `emoji_name`
  * `available`

