$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$toolsDir = Join-Path $repoRoot ".codex-tools"
$tfDir = Join-Path $toolsDir "terraform"
New-Item -ItemType Directory -Force -Path $tfDir | Out-Null

function Get-TerraformVersion {
  $info = Invoke-RestMethod "https://checkpoint-api.hashicorp.com/v1/check/terraform"
  if (-not $info.current_version) { throw "Unable to determine latest terraform version from checkpoint API." }
  return $info.current_version
}

function Ensure-Terraform {
  $ver = Get-TerraformVersion
  $installDir = Join-Path $tfDir $ver
  $tfExe = Join-Path $installDir "terraform.exe"
  if (Test-Path $tfExe) { return $tfExe }

  New-Item -ItemType Directory -Force -Path $installDir | Out-Null
  $zipName = "terraform_${ver}_windows_amd64.zip"
  $zipPath = Join-Path $installDir $zipName
  $url = "https://releases.hashicorp.com/terraform/$ver/$zipName"

  Write-Host "Downloading Terraform $ver ..."
  Invoke-WebRequest -Uri $url -OutFile $zipPath
  Expand-Archive -Path $zipPath -DestinationPath $installDir -Force
  if (-not (Test-Path $tfExe)) { throw "Terraform download did not produce terraform.exe at $tfExe" }
  return $tfExe
}

function Ensure-Go {
  # 1) Prefer system Go (PATH)
  $cmd = Get-Command go -ErrorAction SilentlyContinue
  if ($cmd -and $cmd.Source) { return $cmd.Source }

  # 2) Check codex tools cache (if present)
  $codexGoRoot = Join-Path $repoRoot ".codex-tools\\go"
  if (Test-Path $codexGoRoot) {
    $candidate = Get-ChildItem $codexGoRoot -Recurse -Filter go.exe -ErrorAction SilentlyContinue |
      Where-Object { $_.FullName -match "\\\\go\\\\bin\\\\go\\.exe$" } |
      Sort-Object FullName -Descending |
      Select-Object -First 1
    if ($candidate) { return $candidate.FullName }
  }

  # 3) Check sibling .tmp-go cache (common in this workspace)
  $parent = Split-Path -Parent $repoRoot
  $tmpGoRoot = Join-Path $parent ".tmp-go"
  if (Test-Path $tmpGoRoot) {
    $candidate = Get-ChildItem $tmpGoRoot -Recurse -Filter go.exe -ErrorAction SilentlyContinue |
      Where-Object { $_.FullName -match "\\\\go\\\\bin\\\\go\\.exe$" } |
      Sort-Object FullName -Descending |
      Select-Object -First 1
    if ($candidate) { return $candidate.FullName }
  }

  throw "Go toolchain not found. Install Go or add it to PATH."
}

$tfExe = Ensure-Terraform
$goExe = Ensure-Go

if (-not $env:TF_ACC) { $env:TF_ACC = "1" }
if (-not $env:DISCORD_TOKEN) { throw "DISCORD_TOKEN must be set." }
if (-not $env:DISCORD_GUILD_ID -and -not $env:DISCORD_SERVER_ID) { throw "DISCORD_GUILD_ID (or DISCORD_SERVER_ID) must be set." }

# Put terraform on PATH for terraform-plugin-sdk acceptance test runner.
$env:PATH = (Split-Path -Parent $tfExe) + ";" + $env:PATH

Push-Location $repoRoot
try {
  & $tfExe version
  # Limit parallelism to reduce memory pressure on Windows dev machines.
  $env:GOMAXPROCS = "1"
  $env:GOGC = "25"
  & $goExe test -p 1 ./discord -tags=acctest -run TestAcc -v -timeout 120m
} finally {
  Pop-Location
}
