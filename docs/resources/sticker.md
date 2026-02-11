# Discord Sticker Resource

Manages a guild sticker.

Note: sticker assets are create-only (`file_path` is `ForceNew`).

## Example Usage

```hcl-terraform
resource "discord_sticker" "wave" {
  server_id    = var.server_id
  name         = "wave"
  description  = "Hello"
  tags         = "wave"
  file_path    = "${path.module}/wave.png"
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `name` (Required) Sticker name
* `description` (Optional) Sticker description
* `tags` (Required) Comma-separated emoji names used for sticker search
* `file_path` (Required, ForceNew) Sticker file path
* `reason` (Optional) Audit log reason

## Attribute Reference

* `format_type` Sticker format type

