# Discord Server Data Source

Fetches a server's information.

## Example Usage

```hcl-terraform
data discord_server discord_api {
    server_id = "81384788765712384"
}

output discord_api_region {
    value = data.discord_server.discord_api.region
}
```

## Argument Reference

One of these is required

* `server_id` (Optional) The server id to search for
* `name` (Optional) The server name to search for

## Attribute Reference

* `id` The id of the server
* `region` Region of the server 
* `default_message_notifications` Whether the server has default_message_notifications set to just mentions 
* `verification_level` Required verification level of the server 
* `explicit_content_filter` Explicit Content Filter level of the server 
* `afk_timeout` The AFK timeout of the server
* `afk_channel_id` The AFK channel ID
* `icon_hash` The hash of the server icon
* `splash_hash` The hash of the server splash
* `owner_id` The ID of the owner 

Note: system channel configuration is exposed via the separate `discord_system_channel` data source/resource.
