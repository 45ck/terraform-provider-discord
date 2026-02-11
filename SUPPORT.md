# Support

## Where to Ask for Help

* GitHub Issues: bugs and actionable feature requests

## What to Include

For bug reports, include:

* Terraform CLI version
* Provider version (from `terraform providers` or lockfile)
* Minimal config (redact tokens)
* Error output and (if possible) the Discord API route involved

## Scope / Expectations

This provider is best-effort community maintained. Discord changes frequently; if a specific endpoint is missing,
use the escape hatches:

* `discord_api_request` (generic GET)
* `discord_api_resource` (generic CRUD)
* `discord_guild_settings` (guild PATCH)

