# Fagan-Style Inspection Addendum (Threads + Media Assets Batch)

Date: 2026-02-11

## Scope

This addendum inspects the Framework ports and supporting client changes staged for commit after `594f2e8`:

* Resources:
  * `discord_thread`
  * `discord_thread_member`
  * `discord_stage_instance`
  * `discord_sticker`
  * `discord_soundboard_sound`
  * `discord_server` (adopt + PATCH)
* REST client:
  * Global rate limit coordination added to multipart requests (`DoMultipartWithReason`)

## Findings (Highest Severity First)

1. **Thread “initial message” is create-only; updates must not imply message edits**
   * File: `internal/fw/res_thread.go`
   * Behavior: `content`/`embed` are used only during thread creation.
   * Risk: Users may assume updating these fields edits the initial thread message. Discord does not expose a simple “thread starter message edit” for all thread types via the thread channel itself.
   * Recommendation: Keep `RequiresReplace()` semantics on the `embed` (and create-only `content`) to avoid misleading behavior.

2. **Write-only media inputs should be treated as sensitive and drift-tolerant**
   * Files:
     * `internal/fw/res_sticker.go` (`file_path`)
     * `internal/fw/res_soundboard_sound.go` (`sound_file_path`)
     * `internal/fw/res_server.go` (`icon_data_uri`, `splash_data_uri`)
   * Behavior: Discord does not return the uploaded bytes; state must rely on config for stability.
   * Risk: Operators can’t “repair” drift caused by out-of-band uploads without reapplying config.
   * Recommendation: This is acceptable. Ensure docs make it clear these are write-only.

3. **Multipart requests must honor global rate limits**
   * File: `discord/rest_client_multipart.go`
   * Behavior: global rate limit waiting + cooldown propagation added to multipart retries.
   * Risk: Without this, sticker uploads can cause cascading 429s when other routes are globally limited.
   * Recommendation: Keep this change; it aligns multipart behavior with JSON requests.

4. **`discord_server` overlaps with `discord_guild_settings`**
   * Files:
     * `internal/fw/res_server.go`
     * `internal/fw/res_guild_settings.go`
   * Risk: Two resources PATCH `/guilds/{id}` and can overwrite each other’s fields if both manage the same guild.
   * Recommendation:
     * Prefer `discord_guild_settings` for advanced or rarely-used guild fields.
     * Treat `discord_server` as a convenience wrapper for the common subset.

## Verification Performed (Local)

* `go test -p 1 ./...`
* `go vet ./...`
* `go build -p 1 ./...`

Acceptance tests were not executed (require live Discord credentials + a real guild).

## Follow-Up Work Items

* Add acceptance coverage for at least one of:
  * `discord_thread` creation (forum thread with initial message)
  * `discord_sticker` (multipart upload)
* Consider adding plan-time validation for snowflake-shaped IDs in new resources.

