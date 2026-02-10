# Fagan-Style Inspection Addendum (Governance + Events Batch)

Date: 2026-02-10

## Scope

This addendum inspects the Framework ports added in commit `1fbf4e5`:

* `discord_automod_rule`
* `discord_member_verification` (membership screening / rules gate)
* `discord_onboarding`
* `discord_scheduled_event`

## Findings (Highest Severity First)

1. **Import formats must be explicit for guild-scoped IDs**
   * Files:
     * `internal/fw/res_automod_rule.go`
     * `internal/fw/res_scheduled_event.go`
   * Behavior: Discord requires a guild context for reads (`/guilds/{guild}/.../{id}`), so the resources implement `server_id:<id>` import IDs.
   * Risk: Users coming from the legacy SDK resources (which often used passthrough ID) may assume `terraform import ... <id>` works.
   * Recommendation:
     * Document import formats in README and/or resource docs.
     * Consider supporting both formats when possible (not possible here without an endpoint to map rule/event ID -> guild ID).

2. **JSON passthrough resources are correct but need strong “contract” guidance**
   * Files:
     * `internal/fw/res_automod_rule.go`
     * `internal/fw/res_onboarding.go`
     * `internal/fw/res_member_verification.go`
   * Behavior: `payload_json` is validated as syntactic JSON and normalized at plan-time for diff stability; `state_json` is a normalized snapshot from Discord.
   * Risk:
     * Users may expect `payload_json == state_json`. Discord often injects defaults, reorder arrays, and may omit write-only fields.
     * Applying `payload_json` that includes unknown/unsupported keys will fail at apply-time, not plan-time.
   * Recommendation:
     * Keep this pattern (it is the right tradeoff for evolving schemas), but add a short “how to author payloads” note:
       * Use `discord_api_request` data source to fetch current shapes.
       * Start from `state_json`, then edit.

3. **`discord_scheduled_event` image semantics are write-only**
   * File: `internal/fw/res_scheduled_event.go`
   * Behavior: `image_data_uri` is not readable and will not drift-detect; `image_hash` is computed from Discord.
   * Risk: External changes to the cover image will not be corrected by Terraform unless config changes.
   * Recommendation:
     * This is acceptable. If we want stronger behavior, add a separate “image_hash desired” mechanism (force update when mismatch) but that requires user-provided hash.

4. **`discord_onboarding` and `discord_member_verification` Delete is “disable”, not “restore”**
   * Files:
     * `internal/fw/res_onboarding.go`
     * `internal/fw/res_member_verification.go`
   * Behavior: Delete performs a best-effort disable (`enabled=false`) rather than attempting to revert prior configuration.
   * Risk: Users expecting a full rollback won’t get it.
   * Recommendation:
     * Keep this behavior, but ensure docs clearly state it.

## Discord API Notes

* Audit log reason header is wired for all write calls; Discord ignores it on routes that don’t support it.
* Event status transitions are exposed via the `status` field on update. Discord constrains which transitions are allowed; expect apply-time validation errors from Discord when invalid.

## Verification Performed (Local)

* `go test -p 1 ./...`
* `go vet ./...`
* `go build -p 1 ./...`

Acceptance tests were not executed here (require live Discord credentials + a real guild).

## Follow-Up Work Items (To Reach “No ClickOps”)

* Port remaining JSON-only / REST-only resources that unlock full server governance:
  * threads + thread members
  * stage instances
  * server (guild) resource consolidation (or expand `discord_guild_settings` patterns)
* Add multipart support in `discord.RestClient` so we can implement:
  * stickers
  * soundboard sounds
  * server icon/banner/splash uploads

