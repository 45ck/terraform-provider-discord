# Fagan-Style Inspection Addendum (2026-02-10): Framework Migration + "No Clickops" Coverage

## Trigger

After the initial inspection and the "all REST" refactor, this fork migrated to `terraform-plugin-framework` and removed the legacy SDK + `terraform-plugin-mux` plumbing.

This addendum reviews the migration state, the new `discord_guild_settings` Framework resource, and the provider's ability to manage a guild with minimal "clickops".

## Viewpoint A: Terraform Provider Engineer

### Strengths

* Plugin Framework provides better typing and plan-time hooks (plan modifiers, validators) for long-term maintainability.
* JSON normalization plan modifiers reduce perpetual diffs for "escape hatch" JSON inputs.
* Escape hatches (`discord_api_request`, `discord_api_resource`, `discord_guild_settings`) ensure coverage doesn't lag Discord feature rollout.

### Defects / Risks (Prioritized)

1. Partial Framework migration increases cognitive load and doubles the surface area for schema drift.
   * Risk: during migration, duplicate implementations can drift and cause hard-to-debug planning/apply issues.
   * Mitigation:
     * Migrate resources in larger batches to reduce the "mixed world" duration.

2. Framework resources must remove themselves from state on 404.
   * Fix implemented for `discord_guild_settings`: Framework read removes state when guild returns 404.
   * Apply this pattern to every Framework resource as they migrate.

3. Write-only fields require explicit diff control.
   * Fix implemented for `discord_guild_settings.reason` using an ignore-changes plan modifier.

4. Acceptance tests currently target the SDK provider harness.
   * If/when resources migrate to Framework, acceptance tests should move to `terraform-plugin-testing` with `ProtoV6ProviderFactories`.

## Viewpoint B: Discord API Engineer

### Strengths

* Global rate limit coordination exists in the REST client.
* Audit log reason support is available on the REST layer and is now exposed on `discord_guild_settings`.
* IDs are treated as the stable identifiers; name-based member lookups are disallowed.

### Defects / Risks (Prioritized)

1. Discord endpoints have inconsistent eventual consistency and permissions requirements.
   * Recommendation:
     * Maintain bounded retries for reads after create/update where Discord is known to be eventually consistent.
     * Document which resources require "Manage Guild", "Manage Channels", etc.

2. "Full server control" is realistically bounded by Discord's own model.
   * You cannot create Discord users; membership depends on external join flows.
   * Some operations require the bot to be present in the guild and have specific permissions.

## Reconciled Conclusion

The provider is structurally capable of "no clickops" control of a guild:

* First-class resources cover the most common admin workflows (channels, roles, permissions, messages, automod, onboarding, welcome screen, etc.).
* Escape hatches ensure there are no practical feature limits for JSON REST endpoints.

However, the Framework migration should be treated as a structured project:

* Avoid duplicate implementations for long.
* Keep acceptance tests using `terraform-plugin-testing` with `ProtoV6ProviderFactories`.

## Post-Inspection Actions (Implemented)

* Migrated to a Framework-only provider (protocol v6) and removed the legacy SDK + mux.
* Added a Framework `discord_guild_settings` resource.
* Added:
  * JSON validation for `payload_json`
  * `reason` support (audit log reason)
  * Correct 404 -> remove-from-state behavior
* Improved REST `User-Agent` to include provider version (set from `main`).
* Added examples:
  * `examples/guild_settings`
  * `examples/api_resource_widget`
