# Docker dev/test stack

This directory builds the GeoServer container the project's integration tests run against. It is a **dev/test stack only** — do not use this image in production. The admin password is the literal string `geoserver`, the container ships without TLS, and JVM/resource limits are tuned for a fast `compose up` rather than a hardened deploy.

## What's in the image

| Layer | Choice | Why |
|---|---|---|
| Base | `tomcat:9-jdk17-temurin` | GeoServer 2.x uses the `javax.*` servlet namespace (Servlet 4.x) and does not run on Tomcat 10/11 (which moved to `jakarta.*`). GeoServer 3.0 will be the Tomcat 11 / Jakarta EE target — see [`../docs/version-compat.md`](../docs/version-compat.md). |
| GeoServer | 2.28.0 by default; 2.27.4 LTS for the LTS leg | The supported matrix. CI runs both legs on every PR. Override with `--build-arg GEOSERVER_VERSION=...` or `GEOSERVER_VERSION=2.27.4 make compose-test-up`. |
| Layout | WAR pre-extracted into `webapps/geoserver/` | Lets Tomcat boot faster on subsequent runs. |
| Healthcheck | `curl -fsS http://localhost:8080/geoserver/web/` every 30s after a 120s start period | Compose `depends_on: condition: service_healthy` waits on this before starting the test runner. |

No Importer or Monitor extensions — gismanager only exercises the publish flow (workspace + datastore + feature type), not the import or monitoring REST surfaces. The lower-level [`hishamkaram/geoserver`](https://github.com/hishamkaram/geoserver) Go client's integration suite has those.

## Boot the stack

From the repo root:

```bash
make compose-test-up                          # default — GeoServer 2.28.0
GEOSERVER_VERSION=2.27.4 make compose-test-up # LTS leg
make test-integration                         # runs go test -tags=integration ./...
make compose-test-down
```

PostGIS exposes port `5436` on the host (mapped from the container's `5432`) so it doesn't collide with a local Postgres install or the lower-level `geoserver` client's compose stack. Default credentials: `golang` / `golang`, database `gis`.

Inside the test-runner container, the postgis hostname is just `postgis:5432` — they share the compose network.

## Files in this directory

| File | Role |
|---|---|
| `Dockerfile` | Assembles the GeoServer image. Pulls the GeoServer WAR from `downloads.sourceforge.net` over TLS, unpacks the WAR. |
| `env/geoserver.env` | Environment variables consumed by the container at start: `GEOSERVER_ADMIN_PASSWORD`, JVM `INITIAL_MEMORY` / `MAXIMUM_MEMORY`, plus a few feature toggles (`ENABLE_JSONP`, `MAX_FILTER_RULES`, `OPTIMIZE_LINE_WIDTH`, `XFRAME_OPTIONS`). |
| `postgis/init/01-gismanager.sql` | Idempotently `CREATE EXTENSION IF NOT EXISTS postgis` on the `gis` database. The postgres image runs `*.sql` files from `/docker-entrypoint-initdb.d/` alphabetically on first boot of an empty data volume. To re-run after a schema change, recreate the volume: `docker compose -f docker-compose.test.yml down -v && make compose-test-up`. |

## Production caveat

Do not deploy this image as-is. To use GeoServer in production, build your own image (or use the official one from the GeoServer project) with at minimum: a real admin password, TLS in front, JVM limits tuned for your traffic, persistent volumes for the data dir, and CORS / CSRF configured for your tenancy model.

## See also

- [`../docker-compose.test.yml`](../docker-compose.test.yml) — the integration stack that wires this image together with PostGIS + a test-runner.
- [`../docs/version-compat.md`](../docs/version-compat.md) — supported Go × GeoServer × GDAL × PostGIS matrix and the Tomcat 9 / JDK 17 rationale.
- [`../CONTRIBUTING.md`](../CONTRIBUTING.md) — how to run lint, unit, and integration tests locally.
