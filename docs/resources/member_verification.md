# Discord Member Verification Resource

Manages Membership Screening (the rules gate users must accept before chatting).

This resource uses `payload_json` passthrough because the schema is deep and not currently
covered by the provider's structured resources.

## Example Usage

```hcl-terraform
resource discord_member_verification gate {
  server_id = var.server_id
  payload_json = jsonencode({
    enabled = true
    form_fields = [
      {
        field_type = "TERMS"
        label = "Rules"
        description = "Be respectful."
        required = true
        values = ["I agree"]
      }
    ]
  })
}
```

## Argument Reference

* `server_id` (Required) Server ID
* `payload_json` (Required) JSON payload sent to Discord

## Attribute Reference

* `state_json` Normalized JSON returned by Discord

