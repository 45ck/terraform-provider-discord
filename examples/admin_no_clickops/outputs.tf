output "server_id" {
  value = discord_server.this.server_id
}

output "rules_channel_id" {
  value = discord_channel.rules.id
}

output "rules_webhook_url" {
  value = discord_webhook.rules.url
}

