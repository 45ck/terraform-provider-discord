# Discord Soundboard Default Sounds Data Source

Lists Discord's default soundboard sounds.

## Example Usage

```hcl-terraform
data discord_soundboard_default_sounds defaults {}
```

## Attribute Reference

* `sound` List:
  * `sound_id`
  * `name`
  * `volume`
  * `emoji_id`
  * `emoji_name`
  * `available`

