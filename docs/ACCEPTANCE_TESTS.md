# Acceptance Tests

Acceptance tests exercise real Discord API behavior against a real guild.

They are **opt-in** and require credentials and a test guild.

## Requirements

* Terraform CLI on PATH
* Go toolchain (or use the repo-local Go in `.codex-tools/go/...` if you follow the scripts below)
* A Discord bot token with sufficient permissions in the target guild

## Environment Variables

* `TF_ACC=1` required to enable acceptance tests
* `DISCORD_TOKEN` required
* `DISCORD_GUILD_ID` required (or legacy `DISCORD_SERVER_ID`)

## Running (Windows / PowerShell)

This script downloads a Terraform CLI (if needed) and runs `go test` with the `acctest` build tag:

```powershell
.\scripts\testacc.ps1
```

## Running (Manual)

```powershell
$env:TF_ACC="1"
$env:DISCORD_TOKEN="..."
$env:DISCORD_GUILD_ID="123..."
go test ./discord -tags=acctest -run TestAcc -v -timeout 120m
```

