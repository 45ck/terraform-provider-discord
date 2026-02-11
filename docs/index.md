# Discord Provider

This is a fork of [aequasi/terraform-provider-discord](https://github.com/aequasi/terraform-provider-discord). We ran into some problems with this provider and decided to fix them with this opinionated version.

The Discord provider is used to interact with the Discord API. It requires proper credentials before it can be used.

Use the navigation on the left to read more about the resources and data sources.

## Example Usage

```hcl-terraform
provider "discord" {
  token = var.discord_token
}

data "discord_local_image" "logo" {
  file = "logo.png"
}

resource "discord_server" "my_server" {
  # The provider cannot create guilds with bot tokens. Create the server out-of-band,
  # then import it and manage it via Terraform.
  server_id = var.discord_guild_id
  name      = "My Awesome Server"

  default_message_notifications = 0
  icon_data_uri                 = data.discord_local_image.logo.data_uri
}
```

## Argument Reference

The Discord provider supports the following arguments:

* `token` - The token of the bot that will be accessing the API
* `client_id` - Currently unused
* `secret` - Currently unused
