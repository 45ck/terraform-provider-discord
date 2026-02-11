# Fagan Inspection: Admin Gaps (Widget + Templates) + Audit Reason/Validation

Date: 2026-02-11

## Scope

In-scope:

* New admin-focused resources:
  * `discord_widget_settings` (GET/PATCH `/guilds/{guild.id}/widget`)
  * `discord_guild_template` (CRUD `/guilds/{guild.id}/templates`)
  * `discord_guild_template_sync` (action-style sync via PUT `/guilds/{guild.id}/templates/{code}`)
* Hardening changes:
  * Add `reason` (audit log) support to `discord_channel` and `discord_role`
  * Add cross-field validation for `emoji_id` vs `emoji_name` in:
    * `discord_channel` forum tags + default reaction
    * `discord_soundboard_sound`
    * `discord_welcome_screen`
* Docs and examples updates related to the above.

Out-of-scope:

* Discord endpoints that are not realistically modelable as a stable Terraform resource (pure actions like prune), except where explicitly implemented as action-style resources.

## Artifacts Reviewed

* Commits:
  * `37b8f7e` feat: add widget settings + guild templates resources
  * `4425b9d` hardening: audit-log reason + emoji field validation
* Provider registration: `internal/fw/provider.go`
* New resources:
  * `internal/fw/res_widget_settings.go`
  * `internal/fw/res_guild_template.go`
  * `internal/fw/res_guild_template_sync.go`
* Hardening:
  * `internal/fw/res_channel.go`
  * `internal/fw/res_role.go`
  * `internal/fw/res_welcome_screen.go`
  * `internal/fw/res_soundboard_sound.go`
* Docs/examples:
  * `README.md`
  * `examples/admin_no_clickops/main.tf`

## Entry Criteria

* `go test ./...` passes
* `go vet ./...` passes
* `go build ./...` passes
* Acceptance tests remain opt-in (`TF_ACC=1`) and are not enabled by default

## Inspection Checklist (Results)

* API layer consistency (REST-only): Pass
* Resource schema matches API constraints (basic): Pass, with new cross-field validators
* Import ergonomics documented: Pass (README updated with composite ID formats)
* Backward compatibility impact: Low (new attributes are optional; new resources additive)
* Drift behavior understood and documented: Partial (see open issues)
* Security: Pass (no secrets committed; audit reasons marked Sensitive)

## Findings

### 1. Drift / Destroy Semantics For Settings Resources (Medium)

`discord_widget_settings` is implemented as state-only on destroy (no revert), matching `discord_guild_settings` behavior.

Risk:

* Users may expect that destroying the resource disables the widget (reverting to a safe baseline), but it will only remove from state.

Disposition:

* Accepted for now: avoiding unintentional destructive changes is safer by default.
* Future improvement option:
  * Add an opt-in `revert_on_destroy` attribute (default `false`), which when `true` PATCHes `enabled=false` on Delete.

### 2. Action-Style Terraform Resource: Template Sync (Medium)

`discord_guild_template_sync` is an action-style resource intended to be triggered via `sync_nonce`.

Risk:

* Action resources can surprise users because they do not model a stable remote object.

Disposition:

* Accepted as an explicit opt-in resource with a clear trigger (`sync_nonce`) and no-op destroy.
* Documented inline in README as “sync action resource”.

### 3. Emoji Field Conflicts Were Previously Runtime Failures (Low)

Several Discord payloads accept either `emoji_id` or `emoji_name` (not both). Previously this was only enforced by Discord at runtime.

Disposition:

* Fixed: added `ValidateConfig` checks to fail fast with attribute-specific errors.

## Rework Performed

* Added first-class resources for widget settings and templates, using the REST client.
* Added audit log `reason` support to channel and role operations (Create/Update/Delete and role reordering PATCH).
* Added cross-field validation for `emoji_id` vs `emoji_name` where Discord requires mutual exclusivity.
* Updated README import docs and admin example to demonstrate widget settings management.

## Follow-Up / Next Improvements

* Consider adding acceptance tests (gated by env vars) for:
  * `discord_widget_settings`
  * `discord_guild_template` + `discord_guild_template_sync`
* Normalize non-ASCII “mojibake” in examples (`tags`, `emoji_name`) to either ASCII-safe values or proper UTF-8 literals to reduce copy/paste failures.
* Evaluate whether other resources should also expose optional `reason` fields (only where Discord supports audit reasons).

