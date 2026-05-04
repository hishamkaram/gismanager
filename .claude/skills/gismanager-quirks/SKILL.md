---
description: Catalog of GDAL-binding + GeoServer wire-format quirks that gismanager works around. Reference content; loaded automatically when working on the publish flow, the OGR ingest, or debugging container-side errors.
when_to_use: User mentions "OGR returned 500", "GeoServer returned 500", "no attributes" error, "Layer.Feature drift", "PG connection string", "zip-slip", "libgdal.so missing", "decompression bomb", "soname mismatch", or is implementing a new file format / driver / wire path.
---

# gismanager quirks

Reference material. Each quirk lists the symptom, the root cause, and the file:line where the workaround lives. Whenever you change any code that talks to GDAL or to GeoServer, scan this list for relevance.

## 1. `Layer.Feature(...)` takes `int64`, not `int`

- **Symptom:** `cannot use index (type int) as type int64 in argument to Layer.Feature` from `lukeroth/gdal`. Build fails on Go 1.13+.
- **Root cause:** Upstream's 2022 API sweep moved feature IDs from `int` to `int64`. The pre-revival code was written against the older signature and never bumped — issue #1.
- **Workaround:** Cast `index` to `int64` at the call site.
- **Where:** `layer.go:172` in `GetFeatures`.

## 2. `mholt/archiver` is deprecated; use stdlib `archive/zip`

- **Symptom:** `(*archiver.Zip).Open` requires a pointer receiver in v3+; older code calling `archiver.Zip.Open` (value receiver) fails to build. Upstream is also flagged DEPRECATED in the README.
- **Root cause:** `mholt/archiver` was deprecated in 2024 in favor of `mholt/archives` (which is itself pre-1.0 as of mid-2026).
- **Workaround:** Drop the dep entirely. gismanager only used the Zip extraction; stdlib `archive/zip` does it cleanly. Custom extractor lives in `internal/zipx/extract.go` with zip-slip rejection (filepath.Rel based, recognized by CodeQL's `go/zipslip` query) and a 2 GiB per-entry size cap (gosec G110).
- **Where:** `internal/zipx/extract.go`; called from `utils.go:104` (`zippedShapeFile`).

## 3. PostGIS feature-type creation requires the table to have non-geometry attributes

- **Symptom:** `POST /workspaces/{ws}/datastores/{ds}/featuretypes` returns `400 "Trying to create new feature type inside the store, but no attributes were specified"` if the target PostGIS table is empty or has only the geometry column.
- **Root cause:** GeoServer's wire-format expects the schema to have at least one attribute besides the geometry column to publish a feature type.
- **Workaround:** gismanager uses OGR's `CopyLayer` to materialize the PostGIS table from the source GIS file, which preserves the source schema (geometry + attributes). The integration suite verifies this by publishing a real GeoJSON with attribute fields. Don't introduce a "publish empty stub" code path without addressing the upstream constraint.
- **Where:** `layer.go::LayerToPostgis` (uses `datasource.CopyLayer`); integration test `publish_integration_test.go::TestPublishGeoJSON_EndToEnd_Integration`.

## 4. v2 GeoServer client has no `Exists` method — use `Get` + `errors.Is(err, geoserver.ErrNotFound)`

- **Symptom:** Pre-revival code called `catalog.WorkspaceExists(name)` / `DatastoreExists(...)`. The v2 client doesn't expose those.
- **Root cause:** Deliberate v2 design — `Get` + sentinel matching is more flexible than a boolean (it preserves the underlying `*geoserver.APIError` for 5xx vs 404 disambiguation).
- **Workaround:** The publish flow in `layer.go` does `c.Workspaces.Get(ctx, name)` first; if it returns `ErrNotFound`, calls `Create`; any other error short-circuits. Same pattern for datastore + feature type.
- **Where:** `layer.go::ensureWorkspace`, `ensureDatastore`, plus the inline featuretype check in `PublishGeoserverLayer`.

## 5. Don't `apt-get install libgdal-dev` on the OSGeo image

- **Symptom:** Runtime crash with `error while loading shared libraries: libgdal.so.34: cannot open shared object file: No such file or directory`.
- **Root cause:** The OSGeo image (`ghcr.io/osgeo/gdal:ubuntu-small-3.12.4`) ships its own GDAL (3.12.4 → `libgdal.so.38`). Ubuntu 24's apt has GDAL 3.6 → `libgdal.so.34`. Installing the apt version on top shadows the bundled headers, so the binary links against the older soname which then doesn't exist at runtime.
- **Workaround:** Don't apt-install anything GDAL-named. The OSGeo image already provides headers at `/usr/include/gdal_*.h` and a pkg-config file at `/usr/lib/x86_64-linux-gnu/pkgconfig/gdal.pc`.
- **Where:** `Dockerfile` apt-install line (commented to flag the trap).

## 6. Go 1.25.x stdlib CVEs (GO-2026-4869 / 4865 / 4603 …)

- **Symptom:** `govulncheck` flags 10+ stdlib vulnerabilities on Go 1.25.3.
- **Root cause:** Upstream Go security cadence — patches land in 1.25.x point releases.
- **Workaround:** Pin `GO_VERSION=1.25.9` in `Dockerfile`. When Go 1.25.10+ ships, bump.
- **Where:** `Dockerfile` ARG `GO_VERSION`.

## 7. golangci-lint v2 needs golangci-lint-action v7

- **Symptom:** CI Lint job fails with `invalid version string 'v2.12.1', golangci-lint v2 is not supported by golangci-lint-action v6`.
- **Root cause:** Action v6 only supports v1.x; v7 is the v2-aware release.
- **Workaround:** Pin the action to `golangci/golangci-lint-action@v7` in `.github/workflows/ci.yml`.
- **Where:** `.github/workflows/ci.yml::Lint`.

## 8. `GOFLAGS=-buildvcs=false` required when go-build runs in a container

- **Symptom:** `error obtaining VCS status: exit status 128` from `go build` in CI.
- **Root cause:** The mounted `.git` directory is owned by a different uid than the runner-user; git refuses to operate on it for security reasons.
- **Workaround:** Set `GOFLAGS=-buildvcs=false` at the workflow `env:` level (the dev image has it baked in via `Dockerfile` ENV; on Actions the setup-go step inherits its own env so we set it again).
- **Where:** `.github/workflows/ci.yml::env`, `.github/workflows/integration.yml::env` (if present).

## When this catalog is most useful

- Implementing a new GIS-format ingest path — scan items 1, 2, 5 before writing the OGR open.
- Debugging an integration-test failure with `unmarshal` / `5xx` / `no attributes` — items 3, 4 are the usual suspects.
- Bumping GDAL or Go in `Dockerfile` — items 5, 6 are the gotchas.
- A CI gate flipping red after a workflow / dependency bump — items 7, 8 cover the recurring traps.
