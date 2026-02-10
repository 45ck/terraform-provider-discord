# Discord Scheduled Event Resource

Manages a guild scheduled event.

## Example Usage

```hcl-terraform
resource discord_scheduled_event weekly {
  server_id            = var.server_id
  name                 = "Weekly Hangout"
  entity_type          = 2
  channel_id           = discord_voice_channel.general.id
  scheduled_start_time = "2026-02-15T20:00:00Z"
  privacy_level        = 2
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `name` (Required) Event name
* `description` (Optional) Event description
* `scheduled_start_time` (Required) RFC3339 timestamp
* `scheduled_end_time` (Optional) RFC3339 timestamp
* `privacy_level` (Optional) Privacy level (default 2)
* `entity_type` (Required) 1=stage, 2=voice, 3=external
* `channel_id` (Optional) Required for stage/voice events
* `location` (Optional) External event location
* `image_data_uri` (Optional) data: URI image
* `status` (Optional) Event status (used on update)

## Attribute Reference

* `image_hash` Hash of the event image (from Discord)

