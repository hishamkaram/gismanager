# `testdata/`

This directory holds two kinds of files:

- **Tracked, hand-curated fixtures** (`*.geojson`, `*.gpkg`, `*.kml`,
  `*.zip`, `*.yml`) committed to the repo for legacy unit tests.
- **Manifest-driven fetched fixtures** under `fetched/` — downloaded on
  demand from upstream sources via `make fetch-testdata`. The fetched
  directory is gitignored; only the manifest, license notes, and this
  README are tracked.

## Fetching

```sh
make fetch-testdata
```

This runs `scripts/fetch-testdata.sh`, which walks every row in
`manifest.sha256`, skips files whose sha256 already matches, and downloads
any missing/stale fixture into `fetched/` with retry + sha256 verification.
Re-running is cheap (zero network on a warm cache).

CI caches `fetched/` keyed on the manifest hash, so a typical PR pays the
network cost only when the manifest changes.

## Adding a new fixture

1. Pick an immutable upstream URL (tagged release on GitHub, Natural Earth
   CDN, OGC reference). Never `master`/`main` raw refs.
2. Download once locally, run `sha256sum <file>`, and add a row to
   `manifest.sha256`:
   ```
   <hash>  <relpath>  <url>
   ```
3. Add a license entry to `LICENSES.md`.
4. Reference the fixture from your test as
   `./testdata/fetched/<relpath>`.

## Layout

```
testdata/
├── README.md              ← this file (tracked)
├── LICENSES.md            ← per-fixture upstream licenses (tracked)
├── manifest.sha256        ← fetched-fixture manifest (tracked)
├── fetched/
│   ├── .gitignore         ← `*` (tracked; ignores everything else)
│   └── ...                ← downloads land here (gitignored)
└── *                      ← legacy hand-curated fixtures (tracked)
```
