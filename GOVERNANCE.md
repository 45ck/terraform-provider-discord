# Governance

This project is maintained as a pragmatic Terraform provider fork intended to enable "no clickops" Discord server administration.

## Decision Making

We use a maintainer-driven model with "lazy consensus":

* Most changes are merged after review by at least one maintainer.
* Breaking changes require an explicit note in `CHANGELOG.md` and maintainers should seek broader feedback (issues/discussion) when feasible.

## Compatibility & Stability

* Prefer additive changes (new resources/attributes) over breaking changes.
* Preserve resource IDs and import formats whenever possible.
* If an endpoint is fast-moving or hard to model, prefer the escape hatches over brittle first-class resources.

## Releases

Releases are cut by maintainers. CI must be green.

