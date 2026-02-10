# Discord Stage Instance Resource

Manages a stage instance (topic + privacy) for a stage channel.

## Example Usage

```hcl-terraform
resource discord_stage_instance live {
  channel_id     = discord_channel.stage.id
  topic          = "Town Hall"
  privacy_level  = 2
}
```

## Argument Reference

* `channel_id` (Required) Stage channel ID
* `topic` (Required) Stage topic
* `privacy_level` (Optional) Privacy level (default 2)
* `send_start_notification` (Optional, ForceNew) Send start notification on create
* `scheduled_event_id` (Optional, ForceNew) Link to a scheduled event

## Attribute Reference

* `server_id` Guild ID

