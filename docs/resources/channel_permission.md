# Discord Channel Permission Resource

A resource to create a Permission Overwrite for a channel

## Example Usage

```hcl-terraform
resource discord_channel_permission chatting {
    channel_id = var.channel_id
    type = "role"
    overwrite_id = var.role_id
    # Prefer *_bits64 for future-proof permissions.
    allow_bits64 = data.discord_permission.chatting.allow_bits64
}
```

## Argument Reference

* `type` (Required) Type of the overwrite, `role` or `user`
* `channel_id` (Required) ID of channel for this overwrite
* `overwrite_id` (Required) ID of user or role for this overwrite
* `allow` (Optional) Permission bits for the allowed permissions on this overwrite. At least one of these two (allow, deny) are required. This is a platform-sized integer and can overflow on 32-bit.
* `deny` (Optional) Permission bits for the denied permissions on this overwrite. At least one of these two (allow, deny) are required. This is a platform-sized integer and can overflow on 32-bit.
* `allow_bits64` (Optional) Allow bitset as 64-bit integer string (decimal or `0x...`). Prefer this for newer high-bit permissions.
* `deny_bits64` (Optional) Deny bitset as 64-bit integer string (decimal or `0x...`). Prefer this for newer high-bit permissions.

## Attribute Reference

* `id` Hash of the channel id, overwrite id, and type
