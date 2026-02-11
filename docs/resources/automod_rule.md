# Discord AutoMod Rule Resource

Manages a Discord Auto Moderation rule.

AutoMod rules have multiple shapes depending on `trigger_type`, so this resource uses
`payload_json` passthrough.

## Example Usage

```hcl-terraform
resource "discord_automod_rule" "block_invites" {
  server_id = var.server_id

  payload_json = jsonencode({
    name = "Block invite links"
    event_type = 1
    trigger_type = 3
    trigger_metadata = {
      regex_patterns = ["discord\\.gg/"]
    }
    actions = [{
      type = 1
      metadata = {}
    }]
    enabled = true
    exempt_channels = []
    exempt_roles = []
  })
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `payload_json` (Required) JSON payload used for create/update

## Attribute Reference

* `state_json` Normalized rule JSON returned by Discord

