# Discord Thread Member Resource

Manages a user's membership in a thread.

Use `user_id = "@me"` to manage the bot's membership (join/leave thread).

## Example Usage

```hcl-terraform
resource discord_thread_member bot_join {
  thread_id = discord_thread.t.id
  user_id   = "@me"
}
```

## Argument Reference

* `thread_id` (Required) Thread ID
* `user_id` (Required) User ID or `@me`
* `reason` (Optional) Audit log reason (not read back)

## Attribute Reference

* `join_timestamp`
* `flags`

