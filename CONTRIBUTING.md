# Contributing

## Development Flow
1. Initialize submodules: `git submodule update --init --recursive`
2. Use the repo-pinned SDLC plugin for planning and review: `./scripts/claude -p "/sdlc:plan"`
3. Keep commits small and scoped.

## Testing
- Run unit tests before opening a PR (if you have Go installed): `go test ./...`
- Acceptance tests are opt-in and must not run in CI by default.

