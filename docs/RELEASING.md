# Releasing

This project publishes the Terraform provider under source address:

* `registry.terraform.io/45ck/discord`

## Prerequisites

* Maintainer access to `45ck/terraform-provider-discord`
* Go toolchain installed
* `goreleaser` installed
* GPG key available for checksum signing

## Release Process

1. Ensure `master` is green (`go test ./...`, `go vet ./...`, `go build ./...`).
2. Update `CHANGELOG.md`:
   * Move release notes from `Unreleased` into a new version section.
3. Create and push a tag:
   * `git tag -a vX.Y.Z -m "vX.Y.Z"`
   * `git push origin vX.Y.Z`
4. Build/publish with GoReleaser:
   * `set GPG_FINGERPRINT=...` (PowerShell: `$env:GPG_FINGERPRINT="..."`)
   * `goreleaser release --clean`

## Notes

* Provider runtime address in `main.go` must stay aligned with the published source address:
  * `registry.terraform.io/45ck/discord`
* If users are migrating from `Chaotic-Logic/discord`, they can update state with:
  * `terraform state replace-provider registry.terraform.io/Chaotic-Logic/discord registry.terraform.io/45ck/discord`

