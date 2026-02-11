# Discord Emoji Resource

Manages a guild emoji.

## Example Usage

```hcl-terraform
resource "discord_emoji" "party" {
  server_id       = var.server_id
  name            = "party"
  image_data_uri  = data.discord_local_image.party.data_uri
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `name` (Required) Emoji name
* `image_data_uri` (Required) Emoji image as data URI (ForceNew)
* `roles` (Optional) Restrict emoji usage to these role IDs

## Attribute Reference

* `managed` Whether the emoji is managed
* `animated` Whether the emoji is animated

