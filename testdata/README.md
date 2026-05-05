# `testdata/` and `testdata-fetched/`

Test fixtures live in two sibling directories:

- **`testdata/`** — tracked, hand-curated fixtures (`*.geojson`, `*.gpkg`,
  `*.kml`, `*.zip`, `*.yml`) committed to the repo for the legacy unit
  tests and existing integration tests. The library's `GetGISFiles` /
  `Walk` helpers recurse into this directory in tests.
- **`testdata-fetched/`** — *gitignored*. Manifest-driven fixtures
  downloaded on demand from upstream sources via `make fetch-testdata`.
  Conversion-feature tests (`ConvertVector`, `ConvertRaster`,
  `ReprojectRaster`) reference fixtures here.

The two directories are deliberately siblings: putting fetched binaries
under `./testdata/` would have broken the legacy `TestGetGISFiles` unit
test (counts files in `./testdata/`) and the `TestPublishAll_EndToEnd`
integration test (publishes everything `./testdata/` walks to).

## Fetching

```sh
make fetch-testdata
```

This runs `scripts/fetch-testdata.sh`, which walks every row in
`testdata/manifest.sha256`, skips files whose sha256 already matches, and
downloads any missing/stale fixture into `testdata-fetched/` with retry +
sha256 verification. Re-running is cheap (zero network on a warm cache).

CI caches `testdata-fetched/` keyed on the manifest hash, so a typical PR
pays the network cost only when the manifest changes.

## Adding a new fetched fixture

1. Pick an immutable upstream URL (tagged release on GitHub, Natural Earth
   CDN, OGC reference). Never `master`/`main` raw refs.
2. Download once locally, run `sha256sum <file>`, and add a row to
   `testdata/manifest.sha256`:
   ```
   <hash>  <relpath>  <url>
   ```
   Where `<relpath>` is interpreted relative to `testdata-fetched/`.
3. Add a license entry to `LICENSES.md`.
4. Reference the fixture from your test as
   `./testdata-fetched/<relpath>`.

## Layout

```
testdata/                  ← legacy hand-curated fixtures (tracked)
├── README.md              ← this file (tracked)
├── LICENSES.md            ← per-fixture upstream licenses (tracked)
├── manifest.sha256        ← fetched-fixture manifest (tracked)
├── faults.zip             ← tracked legacy fixtures
├── neighborhood_names_gis.geojson
├── sample.gpkg
└── ...

testdata-fetched/          ← downloaded fixtures (gitignored)
├── .gitignore             ← `*` (tracked; ignores everything else)
├── ne_110m_admin_0_countries.zip
├── RGB.byte.tif
└── ...
```
