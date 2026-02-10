# Discord Voice Channel Resource

A resource to create a voice channel.

Note: this is a legacy per-type channel resource. Prefer `discord_channel` for full channel coverage
(stage/news, additional fields) and future feature support.

## Example Usage

```hcl-terraform
resource discord_voice_channel general {
  name = "General"
  server_id = var.server_id
  position = 0
}
```

## Argument Reference

* `name` (Required) Name of the channel
* `server_id` (Required) ID of server this channel is in
* `position` (Optional) Position of the channel, 0-indexed
* `bitrate` (Optional) Bitrate of the channel
* `user_limit` (Optional) User Limit of the channel
* `category` (Optional) ID of category to place this channel in
