# Fagan-Style Inspection Addendum (Porting Batch 2)

Date: 2026-02-10

## Scope

This addendum inspects the Framework ports added in commit `f2c2cef`:

* `discord_channel_order`
* `discord_role_order`
* `discord_channel_permission` (single overwrite)
* `discord_invite`
* `discord_member_roles`
* `discord_member_nickname`
* `discord_member_timeout`
* `discord_system_channel`

Goals:

* Terraform correctness: stable IDs, predictable diffs, safe CRUD semantics, good import behavior.
* Discord correctness: API compatibility, audit log reasons, drift semantics that align with Discord behavior.

## Findings (Highest Severity First)

1. **`discord_invite` defaults are implicit and can surprise**
   * File: `internal/fw/res_invite.go`
   * Behavior: the resource defaults `max_age` to 86400 and `max_uses` to 0 on create when unset.
   * Risk: Terraform config that omits these fields still has an opinionated invite lifetime. This is consistent with the legacy resource behavior, but is easy to misinterpret as “Discord default”.
   * Recommendation:
     * Document these defaults in the README and/or schema descriptions.
     * Consider making both attributes required (or defaulted explicitly using Framework defaults) if we want fully explicit, reproducible infra.

2. **`discord_member_roles` is authoritative, but only for roles mentioned in config**
   * File: `internal/fw/res_member_roles.go`
   * Behavior: updates add/remove roles for the member based on the plan, and also remove roles that were previously managed by this resource but are removed from config (legacy behavior). It does not attempt to “freeze” all roles for a member.
   * Risk: Operators might assume “member roles are fully managed”. In reality, unmanaged roles can still be assigned out-of-band without Terraform detecting/removing them.
   * Recommendation:
     * Clarify intent in docs: “manages a set of roles for a member”, not “fully locks member roles”.
     * Optional future enhancement: a stricter mode, e.g. `authoritative = true` meaning “member must have exactly this role set”.

3. **Ordering resources intentionally do not revert on destroy**
   * Files: `internal/fw/res_channel_order.go`, `internal/fw/res_role_order.go`
   * Behavior: `Delete` is a state-only removal with a warning.
   * Risk: None functionally (this is the correct design), but needs to be clearly documented to avoid surprises in ephemeral environments.
   * Recommendation:
     * Keep as-is, but ensure README mentions “destroy does not revert”.

4. **`discord_channel_permission` `type` should be validated**
   * File: `internal/fw/res_channel_permission.go`
   * Behavior: runtime validation exists, but schema does not constrain values.
   * Risk: user error produces apply-time failures rather than plan-time validation.
   * Recommendation:
     * Add a `OneOf("role","user")` validator (case-insensitive) for `type`.

5. **`discord_system_channel` is a thin, correct “guild settings subset”**
   * File: `internal/fw/res_system_channel.go`
   * Behavior: PATCHes guild `system_channel_id`, reads via GET guild, and clears on delete.
   * Risk: Potential overlap/interaction with `discord_guild_settings` (both PATCH `/guilds/{id}`).
   * Recommendation:
     * Document that using both resources simultaneously can lead to last-writer-wins behavior unless fields are disjoint.
     * Optional future: implement `discord_guild_settings` as the “one authoritative guild PATCH” resource and make `discord_system_channel` a convenience wrapper that sets `CustomizeDiff`-like drift behavior (Framework equivalent), or keep both but warn.

## Discord API Notes (Design Alignment)

* Audit log reasons:
  * Most write endpoints here already support `reason` via `DoJSONWithReason`. This is the correct direction; keep making audit reasons consistent across all guild-scoped resources.
* Drift:
  * Ordering reads refresh remote positions while preserving the configured ordering list. This is good for drift detection without re-sorting user config.
  * Member nickname/timeout read what Discord returns. Because Discord can modify these fields asynchronously (e.g. timeout expiration), drift is expected and should be treated as normal.

## Verification Performed (Local)

* `go test -p 1 ./...`
* `go vet ./...`
* `go build -p 1 ./...`

Acceptance tests were not executed in this inspection because they require live Discord credentials and a real guild.

## Follow-Up Work Items

* Add plan-time validators (`type`, numeric snowflake shape validators for IDs).
* Add at least one acceptance test that exercises one of the new resources (invite or channel permission).
* Continue porting remaining legacy resources into Framework:
  * webhooks, emojis/stickers, threads, automod, scheduled events, onboarding, verification/rules screening.

