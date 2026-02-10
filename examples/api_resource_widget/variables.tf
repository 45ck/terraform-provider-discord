variable "discord_token" {
  type      = string
  sensitive = true
}

variable "server_id" {
  type = string
}

variable "widget_channel_id" {
  type        = string
  description = "Channel to use for the server widget."
}

