---
description: Run golangci-lint with autofix where supported, then gofmt and goimports across the module — all inside the dev container. Reports the diff produced and any remaining unfixable findings.
allowed-tools: Bash(make lint) Bash(make lint-fix) Bash(make fmt) Bash(make tidy) Bash(docker compose:*) Bash(git diff --stat) Bash(git diff)
---

Apply automatic lint and format fixes to the working tree. Everything runs inside the dev container — no host-side Go invocations.

Steps:

1. **Run autofix:** `make lint-fix`. This shells `golangci-lint run --fix ./...` inside the container with the project's `.golangci.yml` v2 config.
2. **Format:** `make fmt`. Runs `gofmt -s -w` then `goimports -w -local github.com/hishamkaram/gismanager`.
3. **Show what changed:** `git diff --stat` — concise summary of touched files.
4. **List remaining manual fixes.** Re-run `make lint`. For each remaining finding, format as `file:line — linter: message`. Group by linter.
5. Report verdict: **CLEAN** if no remaining findings, or **NEEDS WORK** with the manual-fix list.

Do NOT commit, push, or open a PR. The user reviews the diff and commits manually.
