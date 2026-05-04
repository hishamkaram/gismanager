<!-- Thanks for opening a PR! Please complete the checklist below. -->

## What changes does this PR introduce?

<!-- A clear summary of the change and the motivation. -->

## Type of change

- [ ] Bug fix
- [ ] New feature (additive)
- [ ] Breaking change
- [ ] Documentation only
- [ ] Build / CI / chore

## Checklist

- [ ] My commits use [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `refactor:`, etc.)
- [ ] `make lint` passes locally (inside the dev container — see `CLAUDE.md`)
- [ ] `make test-unit` passes locally
- [ ] `make test-integration` passes locally if my change touches the publish flow (PostGIS load / GeoServer publish)
- [ ] I added or updated unit tests for the change
- [ ] I updated `CHANGELOG.md` under `## [Unreleased]` if user-visible
- [ ] I did not add new runtime dependencies (or I justified why in the description)
- [ ] I did not add anything that requires GDAL on the host machine — all GDAL work runs inside the Docker dev image (per `CLAUDE.md`)

## Related issues

<!-- e.g. Closes #123, Fixes #456 -->
