# Releasing gismanager

This document is the canonical cut-a-release runbook. Releases are an
explicit human action — neither Claude nor CI ever runs `git tag`
autonomously (per `CLAUDE.md` "Don't auto-tag releases").

## What a release produces

The `.github/workflows/release.yml` workflow runs on every `v*.*.*` tag
push and assembles:

- **Multi-arch Docker image** — `ghcr.io/hishamkaram/gismanager:<vX.Y.Z>`
  and `:latest`, built for `linux/amd64` + `linux/arm64`, cosign-signed
  keyless via the GitHub Actions OIDC identity, with provenance + SBOM
  attestations attached by buildx.
- **Linux amd64 binary tarball** — `gismanager_<vX.Y.Z>_linux_amd64.tar.gz`
  containing `gismanager`, `layerSchema`, and `gisconvert`, each
  reporting the released version via `-version` (the `cmd/internal/cli`
  package is populated at build time via `-ldflags`).
- **`checksums.txt`** — SHA-256 of the tarball, with `checksums.txt.sig`
  and `checksums.txt.pem` from a keyless `cosign sign-blob`.
- **SBOM** — `gismanager_<vX.Y.Z>_sbom.spdx.json`, syft-generated SPDX-JSON.
- **GitHub Release** — body is the matching CHANGELOG stanza (see below).

Linux is the only supported runtime for raw binaries — gismanager links
CGo against `libgdal`, so binaries must run on a host with a compatible
GDAL installed. Users who can't satisfy that should use the Docker image.

## Cut steps (maintainer)

### 0. Pre-conditions

- `master` is green on CI (full matrix).
- You are an admin / maintainer on the `hishamkaram/gismanager` repo.
- Your local checkout is on `master`, fully synced with `origin/master`,
  and clean (`git status` reports nothing).
- You have signing keys configured for `git commit -S` and `git tag -s`
  (the project does not currently *enforce* signed commits but signed
  tags are a strong convention).

### 1. Update CHANGELOG

Move every entry under `## [Unreleased]` into a new
`## [<version>] — <YYYY-MM-DD>` stanza. Keep the `[Unreleased]` heading
in place but empty its body — that's the marker the next cycle starts
populating.

The `release.yml` workflow extracts the `[<version>]` body as the
GitHub Release notes; if missing, it falls back to whatever is left
under `[Unreleased]`. So the consolidation step is what makes the
release page useful.

Commit:

```
git add CHANGELOG.md
git commit -S -m "docs(changelog): consolidate Unreleased into [vX.Y.Z] — YYYY-MM-DD stanza"
git push origin master
```

Wait for CI on `master` to go green. Don't skip this — release.yml
doesn't re-run unit/integration tests.

### 2. Tag

```
git tag -s "vX.Y.Z" -m "Release vX.Y.Z"
git push origin "vX.Y.Z"
```

The push triggers `.github/workflows/release.yml`. Watch it under the
Actions tab — typical end-to-end runtime is 10–15 minutes (multi-arch
buildx is the slow leg via QEMU emulation for arm64).

### 3. Smoke test the release

After the workflow finishes:

- **Docker image:** `docker pull ghcr.io/hishamkaram/gismanager:vX.Y.Z`,
  then `docker run --rm ghcr.io/hishamkaram/gismanager:vX.Y.Z -version`
  — should print `gismanager version=vX.Y.Z commit=… built=…`.
- **Cosign verify image:**
  ```
  cosign verify ghcr.io/hishamkaram/gismanager:vX.Y.Z \
    --certificate-identity-regexp 'https://github.com/hishamkaram/gismanager/.github/workflows/release.yml@refs/tags/vX.Y.Z' \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com
  ```
- **Binary tarball:** download from the Release page, extract,
  `./gismanager -version`.
- **Verify checksums:** `sha256sum -c checksums.txt`.
- **Cosign verify checksums:**
  ```
  cosign verify-blob \
    --certificate checksums.txt.pem \
    --signature checksums.txt.sig \
    --certificate-identity-regexp 'https://github.com/hishamkaram/gismanager/.github/workflows/release.yml@refs/tags/vX.Y.Z' \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com \
    checksums.txt
  ```

### 4. Post-release

Open a tracking issue or PR for next-cycle items if relevant. The next
`[Unreleased]` block accumulates entries until the next cut.

## Maintenance branches

When master diverges from a stable release line — e.g. master is leading
edge of v2 work but v1.x users still need security/CVE patches — the
project keeps a `release/vX.x` long-lived branch (one per major-version
line) for back-porting fixes.

### Current state

- **`master`** — leading edge. As of the v2 restructure (see
  `~/.claude/plans/how-can-we-improve-steady-emerson.md`), master is the
  v2 development line. v1 users should NOT pull from master.
- **`release/v1.x`** — v1.x patch line, branched from `v1.4.1`. Any v1
  patch release (`v1.4.2`, `v1.5.0`, etc.) is cut from this branch.

### CI coverage on maintenance branches

`.github/workflows/{ci,integration,security}.yml` trigger on both
`master` and `release/v*` for `push` and `pull_request` events. PRs
opened against `release/v1.x` get the full matrix (lint + unit +
integration on GeoServer 2.27 LTS + 2.28 stable + Trivy + CodeQL +
govulncheck) just like master PRs.

### Cutting a v1 patch

1. Branch off `release/v1.x`: `git checkout release/v1.x && git pull`.
2. Open a feature branch off that and PR back to `release/v1.x`.
3. After CI green + merge, update `CHANGELOG.md` `[Unreleased]`
   on the maintenance branch into a `[1.4.2]` (or whatever) stanza,
   commit, push.
4. Tag `v1.4.2` from the `release/v1.x` HEAD: `git tag -s v1.4.2 …`
   (or `-a` if no GPG configured), `git push origin v1.4.2`. The
   release.yml workflow runs from the tag's commit (not master) so
   the artifact build uses the maintenance-branch source.
5. Smoke test (same as master release).

### Forward-porting fixes

For fixes that must land on BOTH master and `release/v1.x`, prefer:
- Land on `release/v1.x` first, then cherry-pick to master.
- Or land on master first, then cherry-pick to `release/v1.x`.

The cherry-pick direction depends on how invasive the fix is on master
vs. the maintenance branch. Either way, document the cherry-pick
relationship in the second commit's message (`cherry-picked from <sha>`).

## Yanking a bad release

If a release goes out broken:

1. Delete the GitHub Release (web UI or `gh release delete vX.Y.Z`).
   This removes the binary assets but keeps the tag and the Docker
   image — both are immutable by design.
2. Mark the bad release as "yanked" in `CHANGELOG.md` under a
   short `### Yanked` subsection of the `[<version>]` stanza,
   explaining what was wrong and what users should upgrade to.
3. Do NOT delete the tag — that breaks anyone who pinned to it via
   `go get`. The Go module cache is content-addressed and a deleted
   tag with replaced contents would be a supply-chain hazard.
4. Cut a fresh patch release (`vX.Y.Z+1`) that fixes the issue.

## Why not goreleaser?

The plan considered goreleaser but chose a hand-rolled GitHub Actions
workflow because:

- gismanager links CGo against libgdal, which goreleaser-cross handles
  awkwardly — the cross-compiler images don't bundle GDAL and the GDAL
  ecosystem's preferred distribution path is Docker, not raw binaries.
- The project already has a multi-stage Dockerfile that produces both
  the dev environment and the runtime image. The release workflow
  reuses those stages directly via `buildx --target=runtime` and
  `--target=build`, avoiding a parallel build configuration.
- Multi-arch Docker images via `docker buildx build --platform=...`
  are the 2026-current standard for CGo Go binaries; goreleaser-cross
  multi-arch was not adding signal vs. the cost of a second build path.

If a future maintainer wants to add goreleaser for richer changelog
generation or raw-binary support across more platforms, that's a
welcome follow-up — but the current MVP covers the v1.4 release contract.
