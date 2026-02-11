# Discord Soundboard Sound Resource

Manages a guild soundboard sound.

Note: the sound asset is create-only (`sound_file_path` is `ForceNew`).

## Example Usage

```hcl-terraform
resource "discord_soundboard_sound" "airhorn" {
  server_id        = var.server_id
  name             = "airhorn"
  volume           = 1.0
  sound_file_path  = "${path.module}/airhorn.ogg"
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `name` (Required)
* `volume` (Optional) 0..1 (default 1)
* `emoji_id` (Optional)
* `emoji_name` (Optional)
* `sound_file_path` (Required, ForceNew) Path to sound file (base64 encoded for create)
* `reason` (Optional) Audit log reason

## Attribute Reference

* `available` Whether sound is available

