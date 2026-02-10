# Discord Thread Resource

Manages a thread (including forum/media posts, which are threads).

Note: the initial message (`content` / `embed`) is create-only. If you change it, Terraform will
recreate the thread.

## Example Usage

### Thread Without Message

```hcl-terraform
resource discord_thread t {
  channel_id = discord_text_channel.general.id
  name       = "discussion"
  type       = "public_thread"
}
```

### Forum Post

```hcl-terraform
resource discord_thread post {
  channel_id = var.forum_channel_id
  name       = "Release Notes"
  type       = "public_thread"

  content = "v1.2.3 is live"
}
```

## Argument Reference

* `channel_id` (Required) Parent channel
* `name` (Required) Thread name
* `type` (Optional) `public_thread`, `private_thread`, `announcement_thread`
* `message_id` (Optional) Start from an existing message
* `auto_archive_duration` (Optional) Minutes
* `invitable` (Optional) Private thread invite setting
* `rate_limit_per_user` (Optional) Slowmode
* `archived` (Optional) Archive state
* `locked` (Optional) Lock state
* `applied_tags` (Optional) Tag IDs (forum/media)
* `content` (Optional, ForceNew) Initial message content (forum/media)
* `embed` (Optional, ForceNew) Initial message embed (forum/media)

## Attribute Reference

* `server_id` Guild ID

