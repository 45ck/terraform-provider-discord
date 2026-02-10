# Discord Role Everyone Resource

A resource to manage the `@everyone` role's permissions in a server.

## Example Usage

```hcl-terraform
resource discord_role_everyone everyone {
    server_id = var.server_id
    # Prefer permissions_bits64 for future-proof permissions.
    permissions_bits64 = data.discord_permission.everyone.allow_bits64
}
```

## Argument Reference

* `server_id` (Required) Which server the role will be in
* `permissions` (Optional) The permission bits of the role (platform-sized integer; can overflow on 32-bit)
* `permissions_bits64` (Optional) Permissions as 64-bit integer string (decimal or `0x...`). Prefer this for newer high-bit permissions.
