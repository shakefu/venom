# Venom

Go CLI framework: annotated functions become cobra commands, their parameters
become flags, and `@arg` parameters become positional arguments. `README.md`
is the user guide (annotations, supported types, naming, configuration
resolution, positional args, exit codes, extension hooks, `Build`, version
reporting, troubleshooting).

- Root package `venom` (`github.com/shakefu/venom`) — the runtime: `venom.go`
  (options, `App.Execute`/`App.Build`), `command.go` (cobra tree, flag
  registration, reflective dispatch), `config.go` (viper setup, flag binding,
  value resolution, required-flag checks), `naming.go` (Go name → command
  path, flag, env var, config key), `registry.go` (`Register`, called by
  generated `init()`; panics on positional-arg invariant violations),
  `extension.go` (`WithCobra`/`WithViper`), `version.go`, `types.go`
  (`FuncMeta`, `ExitCoder`).
- `gen/` — code generator: finds `venom.Execute()` and `App.Build()` calls,
  parses the comment annotations, writes `venom_gen.go`. Golden test in
  `gen/testdata/simple/` (`input.go` → `expected_gen.go`); there is no
  `-update` flag, so edit the expected file by hand.
- `cmd/venom/` — the `venom` binary; `venom generate [dir]` wraps
  `gen.Generate`.
- `example/` — four runnable programs (`basic/`, `subcommands/`,
  `extensions/`, `build/`; README "Documentation" describes them), each with
  its `venom_gen.go` committed.
- `venom.allium` — Allium 3 spec of the domain behavior (entities, rules,
  invariants); the source of truth over code and README.
- `script/` — scripts-to-rule-them-all. `.github/workflows/` — CI and
  release.

## Required

- Behavior changes start in `venom.allium`: change the rule, then the code
  and tests, which cite rules by name (`// ... implements rule
  ResolveAppVersion in venom.allium`; `TestBuildReturnsArtifacts verifies
  rule AppBuildSucceeds`).
- Hooks run under `prek` (`script/bootstrap` installs them). The config uses
  prek's `repo: builtin`, so the Python `pre-commit` CLI cannot run it.
  Never `--no-verify`; fix the failure and commit again.
- Conventional commits, enforced by the `conventional-pre-commit` commit-msg
  hook and the `Commits` CI job (`cog check --from-latest-tag`). On push to
  `main`, `release.yml` re-runs Lint, Commits and Test, then `cog bump
  --auto` tags and publishes a GitHub release: `fix:` patch, `feat:` minor,
  `!` or `BREAKING CHANGE` major — the commit type sets the release.
- `lint.yml`, `commits.yml` and `test.yml` run on pull requests (and via
  `workflow_call` from the release), not on branch pushes.
- Regenerating `example/*/venom_gen.go` needs `venom` on `PATH`
  (`go install ./cmd/venom && go generate ./...`); no test checks that they
  are current.

## Test/build commands

Prefer `script/*` over the raw commands they wrap:

```sh
script/bootstrap   # go mod download && prek install
script/lint        # prek run --all-files (gofmt, go vet, go build, hygiene)
script/test        # go test ./...
script/build       # go build ./... && go build -o bin/venom ./cmd/venom
```

## Key libraries

- `spf13/cobra` and `spf13/pflag` — command tree and flags.
- `spf13/viper` — value resolution: flag > env > config file > `@default`.
- `charmbracelet/fang` — styled help, completions, man pages and dispatch in
  `Execute`; `Build` returns the cobra root and viper instance instead.

## Maintaining this index

- Update the affected `AGENTS.md` files in the same change when paths,
  responsibilities, commands, dependencies, or conventions change.
- Keep indexes brief: record semantic entry points and non-obvious
  constraints; link to existing documentation instead of duplicating it.
- Add a directory index only when it provides useful navigation beyond its
  parent; omit generated, vendored, and fixture trees.
- Do not add `CLAUDE.md` files; Claude Code reads `AGENTS.md` natively, and
  a `CLAUDE.md` anywhere in the tree stops every `AGENTS.md` from loading.
