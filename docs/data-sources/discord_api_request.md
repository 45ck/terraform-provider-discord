# Discord API Request Data Source

Generic GET-only data source for Discord API paths not yet modeled by the provider.

Prefer first-class resources when they exist. This is an escape hatch.

## Example Usage

```hcl-terraform
data discord_api_request channels {
  path = "/guilds/${var.server_id}/channels"
}

output channels_json {
  value = data.discord_api_request.channels.response_json
}
```

## Argument Reference

* `path` (Required) API path beginning with `/`
* `query_json` (Optional) JSON object of query parameters

## Attribute Reference

* `response_json` Normalized JSON response

