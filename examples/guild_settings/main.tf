terraform {
  required_providers {
    discord = {
      source  = "45ck/discord"
      version = ">= 0.0.0"
    }
  }
}

provider "discord" {
  token = var.discord_token
}

# This resource is an escape hatch for guild-level settings that don't have
# first-class schema coverage yet. Use it to eliminate "clickops" for server settings.
resource "discord_guild_settings" "this" {
  server_id = var.server_id

  # PATCH /guilds/{server_id}
  # Use jsonencode() to keep this valid and stable.
  payload_json = jsonencode({
    preferred_locale               = "en-US"
    default_message_notifications  = 0
    explicit_content_filter        = 2
  })

  reason = "Terraform: set baseline guild settings"
}


