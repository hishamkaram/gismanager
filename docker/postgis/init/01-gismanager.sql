-- Bootstrap script for the gismanager integration test PostGIS database.
--
-- The postgres image runs every *.sql file in /docker-entrypoint-initdb.d/
-- alphabetically on first boot of an empty data volume. To re-run after a
-- schema change, recreate the volume:
--
--     docker compose -f docker-compose.test.yml down -v
--     docker compose -f docker-compose.test.yml up -d --wait
--
-- The integration tests do their own table creation (or use OGR's
-- CopyLayer to materialize a PostGIS table from a GeoJSON source). All this
-- script does is make sure the PostGIS extension is enabled in the `gis`
-- database; the postgis/postgis image normally does this automatically but
-- we re-assert here in case the image's bootstrap order ever changes.

\connect gis;

CREATE EXTENSION IF NOT EXISTS postgis;
