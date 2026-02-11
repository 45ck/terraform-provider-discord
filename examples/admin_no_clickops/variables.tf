variable "discord_token" {
  type      = string
  sensitive = true
}

variable "server_id" {
  type        = string
  description = "Discord guild/server ID to manage."
}

variable "server_name" {
  type        = string
  description = "Server name to enforce."
  default     = "Managed by Terraform"
}

variable "icon_file" {
  type        = string
  description = "Optional path to a server icon (png/jpg). Leave empty to skip."
  default     = ""
}

variable "sticker_file" {
  type        = string
  description = "Optional path to a sticker asset (png/apng/lottie json). Leave empty to skip."
  default     = ""
}

variable "sound_file" {
  type        = string
  description = "Optional path to a sound file for soundboard. Leave empty to skip."
  default     = ""
}

