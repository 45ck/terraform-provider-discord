# Discord Member Timeout Resource

Manages a member timeout (`communication_disabled_until`).

## Example Usage

```hcl-terraform
resource discord_member_timeout cooldown {
  server_id = var.server_id
  user_id   = "123456789012345678"
  until     = "2026-02-10T22:00:00Z"
  reason    = "Cooldown after spam"
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `user_id` (Required) User ID
* `until` (Required) RFC3339 timestamp. Use `""` to clear.
* `reason` (Optional) Audit log reason (not read back)

