# Discord Guild Template Sync Resource

An action-style resource that syncs an existing template to the current guild state via PUT `/guilds/{guild.id}/templates/{code}`.

This resource does not model a distinct remote object. Use it when you explicitly want Terraform to trigger a template sync.

Import requires a composite ID: `server_id:template_code`.

## Example Usage

```hcl-terraform
resource "discord_guild_template_sync" "this" {
  server_id      = var.server_id
  template_code  = discord_guild_template.this.id

  # Change this value to force a resync.
  sync_nonce = "2026-02-11T00:00:00Z"

  reason = "Sync template snapshot"
}
```

## Argument Reference

* `server_id` (Required) Guild (server) ID.
* `template_code` (Required) Template code to sync.
* `sync_nonce` (Optional) Change this value to force a resync (Update) without replacing the resource.
* `reason` (Optional) Audit log reason (not read back).

## Attribute Reference

* `id` Internal Terraform ID (`server_id:template_code`).
* `updated_at` Template `updated_at` value observed after sync (RFC3339 string).
* `is_dirty` Whether the template is out-of-sync with the current guild configuration.

