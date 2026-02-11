# Discord Member Nickname Resource

Manages a member nickname.

## Example Usage

```hcl-terraform
resource "discord_member_nickname" "nick" {
  server_id = var.server_id
  user_id   = "123456789012345678"
  nick      = "New Nick"
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `user_id` (Required) User ID
* `nick` (Required) Nickname. Use `""` to clear.
* `reason` (Optional) Audit log reason (not read back)

