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

# Generic escape-hatch resource example: manage the Guild Widget settings.
# Endpoint: PATCH /guilds/{guild.id}/widget
resource "discord_api_resource" "guild_widget" {
  id_override = var.server_id

  # No create call; treat this as "manage existing".
  create_method = "SKIP"

  read_path = "/guilds/{id}/widget"

  update_method    = "PATCH"
  update_path      = "/guilds/{id}/widget"
  update_body_json = jsonencode({
    enabled    = true
    channel_id = var.widget_channel_id
  })

  delete_method = "SKIP"
}


