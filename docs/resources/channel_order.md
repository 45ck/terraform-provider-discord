# Discord Channel Order Resource

Applies bulk channel ordering in a guild.

Discord ordering can be flaky if you update channels one-by-one; this resource uses
the bulk endpoint to make ordering deterministic.

## Example Usage

```hcl-terraform
resource discord_channel_order order {
  server_id = var.server_id
  reason    = "IaC ordering"

  channel {
    channel_id = discord_category_channel.info.id
    position   = 0
  }

  channel {
    channel_id = discord_text_channel.rules.id
    position   = 1
    parent_id  = discord_category_channel.info.id
  }
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `channel` (Required) List of channels to enforce ordering for
  * `channel_id` (Required)
  * `position` (Required)
  * `parent_id` (Optional)
  * `lock_permissions` (Optional)
* `reason` (Optional) Audit log reason

