# Discord API Resource

Generic JSON-based escape-hatch resource for Discord API endpoints.

This is powerful and potentially dangerous. Prefer first-class resources whenever possible.

Supports `{id}` placeholder in paths.

## Example Usage

```hcl-terraform
resource "discord_api_resource" "guild" {
  # We just want to manage an existing guild as a singleton.
  # create_method=SKIP means the provider will not call the API on create; it will
  # simply use id_override as the resource ID and start managing it.
  create_method = "SKIP"
  id_override   = var.server_id

  read_path       = "/guilds/{id}"

  update_method   = "PATCH"
  update_path     = "/guilds/{id}"
  update_body_json = jsonencode({
    description = "Managed by Terraform"
  })

  # It's rarely safe to "delete" singleton resources. Skip delete to avoid accidental destruction.
  delete_method = "SKIP"
}
```

## Argument Reference

* `create_method` (Optional) Default `POST`. One of: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `SKIP`.
* `create_path` (Optional) Required when `create_method` is not `SKIP`.
* `create_body_json` (Optional, Sensitive)
* `id_field` (Optional) Default `id`
* `id_override` (Optional) If set, used as resource ID. Required when `create_method=SKIP`.

* `read_path` (Required)
* `read_query_json` (Optional)

* `update_method` (Optional) Default `PATCH`. One of: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `SKIP`.
* `update_path` (Optional) Default `read_path`
* `update_body_json` (Optional, Sensitive)

* `delete_method` (Optional) Default `DELETE`. One of: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `SKIP`.
* `delete_path` (Optional) Default `read_path`
* `delete_body_json` (Optional, Sensitive)

* `reason` (Optional) Audit log reason

## Attribute Reference

* `response_json` Normalized JSON response from read
