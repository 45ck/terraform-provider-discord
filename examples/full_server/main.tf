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

resource "discord_role" "moderator" {
  server_id    = var.server_id
  name         = "Moderator"
  permissions  = data.discord_permission.moderator.allow_bits
  hoist        = true
  mentionable  = true
  position     = 5
  color        = data.discord_color.blue.dec
}

data "discord_permission" "moderator" {
  # Example only; adjust to your needs.
  administrator = "allow"
}

data "discord_color" "blue" {
  hex = "#1e90ff"
}

resource "discord_channel" "info" {
  server_id = var.server_id
  type      = "category"
  name      = "info"
  position  = 0
}

resource "discord_channel" "rules" {
  server_id  = var.server_id
  type       = "text"
  name       = "rules"
  parent_id  = discord_channel.info.id
  position   = 0
  topic      = "Read this first."
  nsfw       = false
}

resource "discord_message" "rules_pinned" {
  channel_id = discord_channel.rules.id
  pinned     = true

  embed {
    title       = "Server Rules"
    description = "1) Be respectful\n2) No spam\n3) Follow Discord ToS"
  }
}

resource "discord_role_everyone" "everyone" {
  server_id = var.server_id
}

data "discord_permission" "rules_readonly" {
  send_messages = "deny"
  add_reactions = "deny"
}

resource "discord_channel_permissions" "rules" {
  channel_id = discord_channel.rules.id

  overwrite {
    type         = "role"
    overwrite_id = discord_role_everyone.everyone.id
    deny         = data.discord_permission.rules_readonly.deny_bits
  }
}

resource "discord_welcome_screen" "main" {
  server_id   = var.server_id
  enabled     = true
  description = "Read #rules and say hi."

  channel {
    channel_id   = discord_channel.rules.id
    description  = "Start here"
    emoji_name   = "📜"
  }
}

