# Discord Channel Resource

A resource to manage a Discord guild channel using the Discord HTTP API directly.

This is a more flexible alternative to the older per-type channel resources
(`discord_text_channel`, `discord_voice_channel`, `discord_category_channel`).

## Example Usage

```hcl-terraform
resource "discord_channel" "rules" {
  server_id = var.server_id
  type      = "text"
  name      = "rules"
  topic     = "Server rules"
}
```

## Argument Reference

* `server_id` (Required) ID of the server this channel is in
* `type` (Required) Channel type. Supported: `text`, `voice`, `category`, `news`, `stage`, `forum`, `media`
* `name` (Required) Channel name
* `reason` (Optional) Audit log reason (not read back)
* `position` (Optional) Channel position
* `parent_id` (Optional) Category ID to place this channel in
* `topic` (Optional) Channel topic (text-like channels)
* `nsfw` (Optional) Whether the channel is NSFW
* `rate_limit_per_user` (Optional) Slowmode in seconds (text-like channels)
* `bitrate` (Optional) Bitrate (voice/stage)
* `user_limit` (Optional) User limit (voice/stage)
* `rtc_region` (Optional) RTC region override (voice/stage)
* `video_quality_mode` (Optional) Video quality mode (voice)
* `default_auto_archive_duration` (Optional) Default auto-archive duration for threads
* `default_thread_rate_limit_per_user` (Optional) Thread slowmode in seconds
* `available_tag` (Optional) Forum/media tags
  * `id` (Optional) Tag ID (leave blank to let Discord create)
  * `name` (Required)
  * `moderated` (Optional)
  * `emoji_id` (Optional)
  * `emoji_name` (Optional)
* `default_reaction_emoji` (Optional) Forum default reaction emoji
  * `emoji_id` (Optional)
  * `emoji_name` (Optional)
* `default_sort_order` (Optional) Forum default sort order
* `default_forum_layout` (Optional) Forum default layout

Note: Discord enforces which fields are valid for a given type; invalid combinations
will error from the API.

Note: for `available_tag` and `default_reaction_emoji`, Discord requires that you set at most one of
`emoji_id` or `emoji_name` for a given object; the provider validates this at plan time.
