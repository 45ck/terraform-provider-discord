# Discord Permission Data Source

A simple helper to get computed bit total of a list of permissions

## Example Usage

```hcl-terraform
data "discord_permission" "member" {
    view_channel     = "allow"
    send_messages    = "allow"
    use_vad          = "deny"
    priority_speaker = "deny"
}
data "discord_permission" "moderator" {
    allow_extends = data.discord_permission.member.allow_bits
    deny_extends  = data.discord_permission.member.deny_bits
    kick_members     = "allow"
    ban_members      = "allow"
    manage_nicknames = "allow"
    view_audit_log   = "allow"
    priority_speaker = "allow"
}
resource "discord_role" "member" {
    // ...
    permissions = data.discord_permission.member.allow_bits
}
resource "discord_role" "moderator" {
    // ...
    permissions = data.discord_permission.moderator.allow_bits
}
resource "discord_channel_permission" "general_mod" {
    type = "role"
    overwrite_id = discord_role.moderator.id 
    allow = data.discord_permission.moderator.allow_bits
    deny = data.discord_permission.moderator.deny_bits
}
```

## Argument Reference

* `allow_extends` (Optional) The permission bits to base the new permission set off of for allow
* `allow_extends_bits64` (Optional) Same as `allow_extends` but as a 64-bit integer string (decimal or 0x...)
* `deny_extends` (Optional) The permission bits to base the new permission set off of for deny
* `deny_extends_bits64` (Optional) Same as `deny_extends` but as a 64-bit integer string (decimal or 0x...)

All of the allowed permission values can be found in [the data source](https://github.com/aequasi/terraform-provider-discord/blob/master/discord/data_source_discord_permission.go#L15-47).
Their allowed values are `allow`, `deny`, and `unset`.

## Attribute Reference

* `allow_bits` The allow permission bits (TypeInt; may overflow on 32-bit platforms for newer high-bit permissions)
* `deny_bits` The deny permission bits (TypeInt; may overflow on 32-bit platforms for newer high-bit permissions)
* `allow_bits64` The allow permission bits as a 64-bit integer string (decimal)
* `deny_bits64` The deny permission bits as a 64-bit integer string (decimal)
