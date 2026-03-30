# Venom

Go CLI framework — declarative CLI generation from annotated functions.

## Commits

Use [conventional commits](https://www.conventionalcommits.org/). Examples:
- `feat: add ErrorCode interface for custom exit codes`
- `fix: apply kebab-case to underscore-split command segments`
- `refactor: unify naming functions across packages`
- `docs: add CLAUDE.md`
- `test: add coverage for required flag env resolution`

## Versioning

Semantic versioning via semantic releases driven by conventional commits. Commit types determine version bumps:
- `fix:` → patch
- `feat:` → minor
- `BREAKING CHANGE` footer or `!` suffix → major

## Pre-commit

Uses `prek` (Rust re-implementation, not the Python `pre-commit` CLI). Do not skip hooks with `--no-verify`. If a hook fails, fix the issue and create a new commit.

## CI/CD

GitHub Actions workflows in `.github/workflows/`:
- **Lint** (`lint.yml`) — runs linters on PRs and pushes
- **Test** (`test.yml`) — runs tests on PRs and pushes
- **Release** (`release.yml`) — semantic release on push to `main`, uses [cocogitto](https://docs.cocogitto.io/) and its GitHub Actions

## Scripts

Uses [scripts-to-rule-them-all](https://github.com/github/scripts-to-rule-them-all) in `script/`:
- `script/bootstrap` — install dependencies
- `script/setup` — set up the project after cloning
- `script/test` — run tests
- `script/build` — build the project
- `script/lint` — run linters

Prefer `script/*` over raw commands (e.g. `script/test` not `go test ./...`).

## Architecture

- Functions become CLI commands, parameters become flags
- Code generator (`cmd/venom`) produces `venom_gen.go` with init() registrations
- Runtime resolves config: CLI flags > env vars > config file > @default > zero value
- Allium spec at `venom.allium` is the source of truth for domain behavior
