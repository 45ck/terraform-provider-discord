terraform {
  required_providers {
    discord = {
      source  = "Chaotic-Logic/discord"
      version = ">= 0.0.0"
    }
  }
}

provider "discord" {
  token = var.discord_token
}

data "discord_local_image" "icon" {
  count = var.icon_file != "" ? 1 : 0
  file  = var.icon_file
}

resource "discord_server" "this" {
  server_id = var.server_id
  name      = var.server_name

  # Typical admin defaults. Adjust as needed.
  default_message_notifications = 1
  verification_level           = 1
  explicit_content_filter      = 2
  afk_timeout                  = 300

  # Write-only upload.
  icon_data_uri = try(data.discord_local_image.icon[0].data_uri, null)
}

resource "discord_channel" "info" {
  server_id = var.server_id
  type      = "category"
  name      = "info"
  position  = 0
}

resource "discord_channel" "rules" {
  server_id = var.server_id
  type      = "text"
  name      = "rules"
  parent_id = discord_channel.info.id
  position  = 0
  topic     = "Read this first."
  nsfw      = false
}

resource "discord_message" "rules_pinned" {
  channel_id = discord_channel.rules.id
  pinned     = true

  embed {
    title       = "Server Rules"
    description = "1) Be respectful\n2) No spam\n3) Follow Discord ToS"
  }
}

resource "discord_thread" "faq" {
  channel_id = discord_channel.rules.id
  type       = "public_thread"
  name       = "FAQ"
}

resource "discord_thread_member" "bot_in_faq" {
  thread_id = discord_thread.faq.id
  user_id   = "@me"
}

resource "discord_webhook" "rules" {
  channel_id = discord_channel.rules.id
  name       = "rules-updater"
}

resource "discord_sticker" "hello" {
  count = var.sticker_file != "" ? 1 : 0

  server_id   = var.server_id
  name        = "hello"
  description = "managed by terraform"
  tags        = "🙂"
  file_path   = var.sticker_file
}

resource "discord_soundboard_sound" "ping" {
  count = var.sound_file != "" ? 1 : 0

  server_id       = var.server_id
  name            = "ping"
  sound_file_path = var.sound_file
  volume          = 1.0
  emoji_name      = "🔊"
}

# Optional: manage the server widget settings (no clickops).
# If you enable it, set channel_id to a public channel.
resource "discord_widget_settings" "this" {
  server_id = var.server_id
  enabled   = false
}