# Discord Guild Settings Resource

Applies arbitrary guild settings via `PATCH /guilds/{guild.id}`.

This is the recommended way to manage guild-level settings that are not yet modeled as
first-class arguments on `discord_server`.

## Example Usage

```hcl-terraform
resource "discord_guild_settings" "main" {
  server_id = var.server_id

  payload_json = jsonencode({
    description = "No clickops."
    preferred_locale = "en-US"
    rules_channel_id = discord_text_channel.rules.id
    public_updates_channel_id = discord_text_channel.announcements.id
  })
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `payload_json` (Required) JSON payload sent to Discord

## Attribute Reference

* `state_json` Normalized JSON returned by Discord

## Destroy Behavior

Destroy is a no-op and will not revert settings. If you need reversions, change `payload_json`
explicitly, or use `lifecycle { prevent_destroy = true }`.

