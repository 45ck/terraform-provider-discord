# Discord Ban Resource

Manages a guild ban.

This is powerful and can be disruptive.

## Example Usage

```hcl-terraform
resource "discord_ban" "spambot" {
  server_id = var.server_id
  user_id   = "123456789012345678"
  delete_message_seconds = 3600
  reason    = "Spam"
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `user_id` (Required) User ID to ban
* `delete_message_seconds` (Optional) How many seconds of messages to delete
* `reason` (Optional) Audit log reason (not read back)

