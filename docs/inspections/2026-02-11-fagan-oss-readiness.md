# Fagan-Style Inspection (2026-02-11): Open Source Readiness

## Objective

Make the repository easier to understand, safer to consume, and easier to contribute to as a maintained community fork.

## Scope

In-scope:

* Repo metadata and contributor experience (`README.md`, `CONTRIBUTING.md`, issue/PR templates)
* Security posture (`SECURITY.md`, vulnerability reporting)
* Governance and bus-factor clarity (`MAINTAINERS.md`, `GOVERNANCE.md`, `SUPPORT.md`)
* CI determinism (pin Go version from `go.mod`)
* Dependency update automation (Dependabot)

Out-of-scope:

* Rewriting provider internals
* Large behavioral changes to provider resources/data sources
* Changing license or module path

## Entry Criteria

* CI passes (`go test`, `go vet`, `go build`, `gofmt`)

## Findings

### 1. Missing Governance and Support Documents (High)

Risk:

* As a fork-of-a-fork, external contributors cannot tell who owns decisions, how to get help, or what the support model is.

Action:

* Added `MAINTAINERS.md`, `GOVERNANCE.md`, `SUPPORT.md`.

### 2. Missing Security Reporting Path (High)

Risk:

* Vulnerabilities may be reported via public issues (token handling, request signing, etc.), increasing exposure.

Action:

* Added `SECURITY.md` and wired issue template contact link to Security Advisories.

### 3. CI Not Deterministic on Go Version (Medium)

Risk:

* Using `go-version: stable` can introduce unexpected breakage when new Go versions are released.

Action:

* Updated CI to use `go-version-file: go.mod`.
* Added `go mod tidy` check and `go build` step.

### 4. Incomplete Docs for New Resources (Medium)

Risk:

* Resources exist but no dedicated docs pages exist; users rely on README only.

Action:

* Added docs pages for: `discord_widget_settings`, `discord_guild_template`, `discord_guild_template_sync`.

## Exit Criteria

* OSS readiness docs exist and CI checks enforce formatting, module tidiness, and buildability.

