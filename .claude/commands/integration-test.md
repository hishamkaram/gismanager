---
description: Boot the GeoServer + PostGIS docker-compose-test stack and run the //go:build integration test suite. Tears down on success; leaves the stack up on failure for inspection.
argument-hint: [version]
allowed-tools: Bash(make compose-test-up) Bash(make compose-test-down) Bash(make compose-test-logs) Bash(make test-integration) Bash(docker compose:*) Bash(GEOSERVER_VERSION=*) Bash(curl:*)
---

Run the gismanager integration suite against GeoServer `$ARGUMENTS` (default: `2.28.0`; pass `2.27` for the LTS leg).

Steps:

1. **Boot the stack.**
   - `2.28` (default) → `make compose-test-up`.
   - `2.27` → `GEOSERVER_VERSION=2.27.4 make compose-test-up`.
2. **Wait for healthcheck.** `make compose-test-up` already passes `--wait`, so it returns when both `geoserver` and `postgis` containers are healthy. First boot of GeoServer takes ~120s while Tomcat unpacks the WAR.
3. **Run the suite:** `make test-integration`. Capture full output.
4. **On success:**
   - `make compose-test-down`.
   - Report green: test count + duration.
5. **On failure:**
   - Print the failed test names (grep for `--- FAIL:` blocks).
   - Dump `docker compose -f docker-compose.test.yml logs --tail=200 geoserver`.
   - Dump `docker compose -f docker-compose.test.yml logs --tail=100 postgis`.
   - **Do NOT tear the stack down** — leave it running so the user can `curl http://localhost:8080/geoserver/web/` or `psql -h localhost -p 5436 -U golang gis`.
   - Stop. Don't propose code edits.
