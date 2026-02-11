# Discord Thread Members Data Source

Lists members in a thread.

## Example Usage

```hcl-terraform
data "discord_thread_members" "members" {
  thread_id = discord_thread.t.id
}

output member_ids {
  value = [for m in data.discord_thread_members.members.member : m.user_id]
}
```

## Argument Reference

* `thread_id` (Required) Thread ID
* `limit` (Optional)
* `after` (Optional) User ID snowflake for pagination
* `with_member` (Optional) Include guild member data where supported

## Attribute Reference

* `member` List of members:
  * `user_id`
  * `join_timestamp`
  * `flags`

