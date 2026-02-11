# Discord Onboarding Resource

Manages the server onboarding configuration.

This resource uses a `payload_json` passthrough because Discord's onboarding schema
is deep and evolves over time.

## Example Usage

```hcl-terraform
resource "discord_onboarding" "main" {
  server_id = var.server_id
  payload_json = jsonencode({
    enabled = true
    default_channel_ids = [discord_text_channel.rules.id]
    prompts = []
  })
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `payload_json` (Required) JSON payload sent to Discord via `PUT /guilds/{guild.id}/onboarding`

## Attribute Reference

* `state_json` Normalized onboarding JSON returned by Discord

