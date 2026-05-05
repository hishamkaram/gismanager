# Testdata licenses

Each row in `manifest.sha256` is a fixture downloaded into the gitignored
`testdata/fetched/` directory at test time. None of the binary fixtures are
distributed with this repository — we only ship the manifest and these
license notes. Licenses below cover the upstream fixtures we *fetch*.

## Vector fixtures

### `ne_110m_admin_0_countries.zip`, `ne_110m_admin_0_countries.geojson`

- **Source:** Natural Earth (https://www.naturalearthdata.com/)
- **License:** Public domain. From the Natural Earth terms: "All versions of
  Natural Earth raster + vector map data found on this website are in the
  public domain. You may use the maps in any manner, including modifying the
  content and design, electronic dissemination, and offset printing. The
  primary authors […] and the Natural Earth contributors release the maps
  for use in any way."
- **Tracking refs:**
  - `.zip` — pulled live from `naciscdn.org` (the project's stable CDN).
  - `.geojson` — pinned to `nvkelso/natural-earth-vector` tag `v5.1.2`
    (https://github.com/nvkelso/natural-earth-vector/tree/v5.1.2). Tagged
    refs are immutable; sha256 in the manifest locks the exact bytes.

## Raster fixtures

### `RGB.byte.tif`

- **Source:** rasterio test suite
  (https://github.com/rasterio/rasterio/tree/1.4.3/tests/data).
- **License:** BSD-3-Clause (rasterio's project license; the test data is
  redistributed under the same terms).
- **Tracking ref:** pinned to `rasterio` tag `v1.4.3`.

### `cog.tif`

- **Source:** rio-tiler test fixtures
  (https://github.com/cogeotiff/rio-tiler/tree/7.0.1/tests/fixtures).
- **License:** BSD-3-Clause (rio-tiler's project license).
- **Tracking ref:** pinned to `rio-tiler` tag `7.0.1`.

## Adding a fixture

1. Pick an immutable URL — a tagged release on GitHub, an OGC reference URL,
   a CDN-served Natural Earth file. Never `master` or `main` raw refs.
2. Confirm the upstream license is OSS-compatible (BSD-2/3, MIT, public
   domain, CC0, CC-BY all fine; ODbL OK with attribution; GPL-only fixtures
   need separate review).
3. Download once locally, compute `sha256sum`, append to `manifest.sha256`.
4. Add a "### `<filename>`" entry to this file with source + license + ref.

If a fixture's upstream URL ever 404s, the fallback is to mirror the
fixture as a release asset on this repository and update the manifest URL —
do not commit the binary into the source tree.
