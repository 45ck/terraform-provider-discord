# Discord Welcome Screen Resource

Manages the server welcome screen for a community-enabled guild.

## Example Usage

```hcl-terraform
resource discord_welcome_screen main {
  server_id    = var.server_id
  enabled      = true
  description  = "Read the rules and say hi."

  channel {
    channel_id  = discord_text_channel.rules.id
    description = "Start here"
    emoji_name  = "📜"
  }
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `enabled` (Optional) Enable welcome screen (default true)
* `description` (Optional) Welcome screen description
* `channel` (Optional) List of welcome channels
  * `channel_id` (Required)
  * `description` (Required)
  * `emoji_id` (Optional)
  * `emoji_name` (Optional)

