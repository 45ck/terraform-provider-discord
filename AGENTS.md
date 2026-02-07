# Repo Instructions (Terraform Discord Provider)

## Ground Rules
- Prefer small PRs. Keep diffs focused (one feature/fix per PR).
- Preserve provider behavior unless the change is explicitly part of the PR scope.
- Never commit secrets. Use `TF_VAR_token` or `.tfvars` that is gitignored for local runs.

## SDLC Plugin (Recommended)
This repo vendors the Claude SDLC plugin as a git submodule so contributors can run a consistent workflow.

- Run Claude with the repo-pinned plugin: `./scripts/claude`
- Typical workflow:
  - `./scripts/claude -p "/sdlc:init"`
  - `./scripts/claude -p "/sdlc:plan"`
  - `./scripts/claude -p "/sdlc:review"`
  - `./scripts/claude -p "/sdlc:qa"`

## Terraform Provider Notes
- Provider auth is configured via the Terraform provider `token` field (required). Prefer `TF_VAR_token` locally.
- Acceptance tests (if/when used) should be opt-in via `TF_ACC=1` and must not run in PR CI by default.

