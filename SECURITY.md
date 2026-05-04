# Security Policy

## Supported versions

| Version | Supported |
|---|---|
| 1.x   | ✅ Active development and security fixes (latest `v1.0.0`, on `master`) |
| < 1.0 | ❌ No support — pre-revival source (last commit October 2018, build broken since Go 1.13). |

## Reporting a vulnerability

**Please do not open public GitHub issues for security vulnerabilities.**

Use one of these private channels:

1. **GitHub Security Advisories** (preferred) — go to the repository's [Security tab](https://github.com/hishamkaram/gismanager/security) → "Report a vulnerability".
2. **Email** — open a private advisory request via GitHub if you cannot use the Security tab.

Please include:

- A description of the vulnerability and its impact.
- A minimal reproduction (config + sample data, or a Go program if using the library directly).
- The version of `github.com/hishamkaram/gismanager`, Go, GeoServer, and GDAL where the issue was observed.

## What to expect

- Acknowledgement within 5 business days.
- A triage response with severity assessment within 10 business days.
- A fix, mitigation, or workaround as quickly as severity allows. Critical issues are prioritized.
- A coordinated disclosure: we will agree on a public disclosure timeline with the reporter before publishing.

## Scope

This policy covers the Go code in this repository (the `gismanager` library + the two CLIs in `cmd/`) and the dev/test container images shipped from `Dockerfile` and `docker/Dockerfile`.

Out of scope:

- **Upstream GeoServer** (the server) — report to the [GeoServer project](https://geoserver.org/) directly.
- **Upstream GDAL / OGR** — report to [OSGeo/gdal](https://github.com/OSGeo/gdal/issues).
- **`lukeroth/gdal` Go bindings** — report to [lukeroth/gdal](https://github.com/lukeroth/gdal/issues).
- **`hishamkaram/geoserver` (the lower-level Go client)** — report to [hishamkaram/geoserver](https://github.com/hishamkaram/geoserver/security).
- **Other dependencies** — please also report upstream so the broader ecosystem benefits.
