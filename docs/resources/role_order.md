# Discord Role Order Resource

Applies bulk role ordering in a guild.

## Example Usage

```hcl-terraform
resource discord_role_order order {
  server_id = var.server_id

  role {
    role_id   = discord_role.moderator.id
    position  = 10
  }
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `role` (Required) List of roles to enforce ordering for
  * `role_id` (Required)
  * `position` (Required)
* `reason` (Optional) Audit log reason

