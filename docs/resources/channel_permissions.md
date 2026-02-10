# Discord Channel Permissions Resource

Manages the full set of permission overwrites on a channel.

This resource is **authoritative**: during apply it will delete any permission overwrites
on the channel that are not declared in this resource.

If you want to manage individual overwrites without taking ownership of the entire set,
use `discord_channel_permission` instead.

## Example Usage

```hcl-terraform
resource discord_channel_permissions perms {
  channel_id = discord_channel.rules.id

  overwrite {
    type         = "role"
    overwrite_id = discord_role.moderator.id
    # Prefer *_bits64 for future-proof permissions.
    allow_bits64 = data.discord_permission.moderator.allow_bits64
  }
}
```

## Argument Reference

* `channel_id` (Required) Channel ID
* `overwrite` (Required) Set of overwrites:
  * `type` (Required) `role` or `user`
  * `overwrite_id` (Required) Role ID or user ID
  * `allow` (Optional) Allow bitset (platform-sized integer; can overflow on 32-bit)
  * `deny` (Optional) Deny bitset (platform-sized integer; can overflow on 32-bit)
  * `allow_bits64` (Optional) Allow bitset as 64-bit integer string (decimal or `0x...`). Prefer this for newer high-bit permissions.
  * `deny_bits64` (Optional) Deny bitset as 64-bit integer string (decimal or `0x...`). Prefer this for newer high-bit permissions.
* `reason` (Optional) Audit log reason (not read back)
