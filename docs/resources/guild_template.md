# Discord Guild Template Resource

Manages a guild template (list/create/update/delete via `/guilds/{guild.id}/templates`).

Templates are guild-scoped objects; import requires a composite ID: `server_id:template_code`.

## Example Usage

```hcl-terraform
resource "discord_guild_template" "this" {
  server_id    = var.server_id
  name         = "Managed Template"
  description  = "Updated by Terraform"
  reason       = "Managed by Terraform"
}
```

## Argument Reference

* `server_id` (Required) Guild (server) ID.
* `name` (Required) Template name.
* `description` (Optional) Template description.
* `reason` (Optional) Audit log reason (not read back).

## Attribute Reference

* `id` Template code.
* `usage_count` Number of times the template was used.
* `is_dirty` Whether the template is out-of-sync with the current guild configuration.
* `created_at` Creation timestamp (RFC3339 string).
* `updated_at` Last update timestamp (RFC3339 string).
* `creator_id` ID of the template creator.

