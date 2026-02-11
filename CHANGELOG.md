# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project aims to follow SemVer where practical for Terraform provider expectations.

## Unreleased

### Added

### Changed

### Fixed

## [0.1.0] - 2026-02-11

### Added

* New first-class resources:
  * `discord_widget_settings`
  * `discord_guild_template`
  * `discord_guild_template_sync` (action-style template sync)
* OSS governance and contributor scaffolding:
  * `SECURITY.md`, `CODE_OF_CONDUCT.md`, `SUPPORT.md`, `MAINTAINERS.md`, `GOVERNANCE.md`
  * GitHub issue/PR templates, CODEOWNERS, Dependabot config

### Changed

* Provider migration and identity alignment:
  * Module path now aligns with this fork: `github.com/45ck/terraform-provider-discord`
  * Provider source/docs/examples aligned to `45ck/discord`
* CI hardening:
  * Deterministic Go setup from `go.mod`
  * Added `go mod tidy` check and `go build` in CI
* Documentation refresh:
  * Updated docs/examples to modern HCL syntax
  * Added docs pages for widget/template resources
  * Added explicit migration guidance from `Chaotic-Logic/discord` to `45ck/discord`

### Fixed

* Added/expanded audit-log `reason` support across key write resources.
* Added stricter cross-field validation for mutually exclusive emoji fields.
