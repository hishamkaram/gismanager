# Version compatibility

This document is the source of truth for what gismanager v1.0.x supports, what's tested in CI, and what's parked.

## Go

| Version | Status | Notes |
|---|---|---|
| **1.25** | **Required (minimum)** | Set by `go.mod`; required transitively by `github.com/hishamkaram/geoserver/v2 v2.0.0+`. |
| 1.26+    | Forward-compatible | No CI gate yet; bump matrix when GA ships. |
| ≤ 1.24   | Unsupported | Won't compile (transitive `geoserver/v2` requires 1.25). |

## GeoServer (the server)

| Version | Status | Tested in CI | Notes |
|---|---|---|---|
| **2.27.4 LTS** | Supported (CI) | `GeoServer 2.27.4` integration job | Long-term support release. |
| **2.28.0 stable** | Supported (CI) | `GeoServer 2.28.0` integration job | Current stable. |
| 2.18 – 2.26 | Best-effort | not in CI | Many endpoints work; security and feature-type-discovery responses may differ in shape. |
| ≤ 2.17 | Unsupported | not in CI | Pre-modern security API; major drift in JSON response shapes. |
| 3.0.x | Tracked for v1.x | not in CI | Jakarta EE / Tomcat 11 / ImageN raster engine. Validates only after the upstream migration settles in production deploys. |

**Integration coverage:** every PR runs the full integration suite against both 2.27.4 LTS and 2.28.0 stable. Both legs must pass before merge. The matrix lives in [`../.github/workflows/integration.yml`](../.github/workflows/integration.yml).

## GDAL

| Version | Status | Notes |
|---|---|---|
| **3.12.4** | Pinned (CI + dev) | `ghcr.io/osgeo/gdal:ubuntu-small-3.12.4` is the base image for both the dev container and the GeoServer container. Bumping requires re-running the integration suite end-to-end. |
| 3.10–3.11 | Best-effort | Likely works (the `lukeroth/gdal` Go bindings target `>=3.x`). Not gated. |
| < 3.10 | Unsupported | Older OGR APIs — `Layer.Feature(int)` vs `Layer.Feature(int64)`, etc. |

**Why pin to one GDAL?** The `lukeroth/gdal` CGo bindings link against system GDAL at compile time; the binary's `libgdal.so.NN` soname is locked to whatever the build container's GDAL ships. Pinning the dev/build image to a specific GDAL minor avoids surprise soname mismatches between a developer's container and the runtime image.

## PostGIS

| Version | Status | Notes |
|---|---|---|
| **PostGIS 16-3.4** (`postgis/postgis:16-3.4`) | Pinned (CI + dev) | What `docker-compose.test.yml` boots. |
| ≥ 2.5 | Best-effort | OGR's PostgreSQL driver supports any modern PostGIS. Older PostGIS lacks GIST index defaults that LayerToPostgis assumes. |
| < 2.5 | Unsupported | |

## Tomcat / Java (for GeoServer)

The integration GeoServer container uses **Tomcat 9 + JDK 17 (Temurin)**. GeoServer 2.x (2.27 / 2.28) is built against the `javax.*` servlet namespace (Servlet 4.x); Tomcat 10+ moved to `jakarta.*` and breaks GeoServer 2.x at WAR-deploy time. GeoServer 3.0 will unblock Tomcat 11 — see the GeoServer roadmap.

## Module path

| Path | Status |
|---|---|
| `github.com/hishamkaram/gismanager` | v1.x — stable (latest `v1.0.0`) |

No `/v2` semantic-import-versioning suffix — gismanager has no released versions to preserve, so v1.0.0 is the first stable tag.

## When this matrix changes

- **Go 1.26 release** → add to the matrix as untested, then move to supported once CI validates.
- **GeoServer 2.29 release** → swap 2.28 for 2.29 in the integration matrix; keep 2.27 LTS as the LTS leg.
- **GeoServer 2.27 LTS retires** → drop from the matrix when GeoServer's own LTS support ends.
- **GeoServer 3.0 stabilizes** → add as a third matrix entry once Jakarta EE / Tomcat 11 / ImageN settle.
- **GDAL 3.13 release** → bump the pin in `Dockerfile` and `docker/Dockerfile`, re-run the integration suite end-to-end on both GeoServer legs before promoting.

## Cross-references

- [`../README.md`](../README.md) — install, quick start
- [`architecture.md`](architecture.md) — package shape, design tenets
