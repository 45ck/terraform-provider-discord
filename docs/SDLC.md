# SDLC Workflow (Claude + Repo-Pinned Plugin)

This repository includes the Claude SDLC plugin as a submodule under `tools/claude-sdlc-plugin`.

## One-Time Setup
```sh
git submodule update --init --recursive
```

## Run Claude With The Repo-Pinned Plugin
```sh
./scripts/claude -p "/sdlc:plan"
```

## Suggested Iteration Loop
1. Plan: `./scripts/claude -p "/sdlc:plan"`
2. Implement: follow the generated tasks and keep commits small.
3. Review: `./scripts/claude -p "/sdlc:review"`
4. QA: `./scripts/claude -p "/sdlc:qa"`

## Verification Expectations
- Unit tests must pass before merge.
- Acceptance tests are opt-in and require explicit credentials; they must not be enabled by default in CI.

