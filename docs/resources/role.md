# Discord Role Resource

A resource to create a role

## Example Usage

```hcl-terraform
resource discord_role moderator {
    server_id = var.server_id
    name = "Moderator"
    # Prefer permissions_bits64 for future-proof permissions.
    permissions_bits64 = data.discord_permission.moderator.allow_bits64
    color = data.discord_color.blue.dec
    hoist = true
    mentionable = true
    position = 5
}
```

## Argument Reference

* `server_id` (Required) Which server the role will be in
* `name` (Required) The name of the role
* `permissions` (Optional) The permission bits of the role (platform-sized integer; can overflow on 32-bit)
* `permissions_bits64` (Optional) Permissions as 64-bit integer string (decimal or `0x...`). Prefer this for newer high-bit permissions.
* `color` (Optional) The integer representation of the role color
* `hoist` (Optional) Whether the role should be hoisted (default false)
* `mentionable` (Optional) Whether the role should be mentionable (default false)
* `position` (Optional) The position of the role. This is reverse indexed (@everyone is 0)

## Attribute Reference

* `managed` Whether this role is managed by another service
