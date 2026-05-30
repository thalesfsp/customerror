# customerror — project guide for Claude

Go library providing rich, structured custom errors (`*CustomError`) with a code,
HTTP status code, tags, fields, i18n/translations, retryability, and an error
`Catalog`. Module path: `github.com/thalesfsp/customerror` (v1, no `/vN` suffix).

## Layout
- `customerror.go` — core `CustomError` type; `Error()/APIError()/JustError()`;
  `MarshalJSON`; `Is`/`Unwrap`; `Copy`; package funcs (`New`, `Factory`, `From`,
  `Wrap`, `To`, `IsCustomError`, `IsHTTPStatus`, `IsErrorCode`, `IsRetryable`);
  instance factory methods (`cE.NewFailedToError/NewInvalidError/NewMissingError/
  NewRequiredError/NewHTTPError/New`, `FormatError`, `NewChildError`).
- `builtin.go` — package-level constructors (`NewFailedToError`, `NewInvalidError`,
  `NewMissingError`, `NewRequiredError`, `NewNotFoundError`, `NewHTTPError`).
- `options.go` — functional options + `prependOptions`.
- `language.go` — `Language` type, ISO 639-1/3166-1 regex, `NewLanguage`.
- `languageprefixtemplate.go` — built-in translation templates (singleton via
  `sync.Once`), `GetTemplate`, `AddNewLanguage`, `NewErrorPrefixMap`.
- `catalog.go` — `Catalog` (error code → error), `ErrorCode` validation.
- Tests: `customerror_test.go`, `catalog_test.go`,
  `languageprefixtemplate_test.go`, `example_test.go` (runnable Examples that
  assert EXACT stdout), `bugfixes_test.go` (regression tests; happy/bad/edge).

## Conventions
- Functional options (Rob Pike style). Options are `func(*CustomError)` and
  cannot return errors, so invalid input is ignored gracefully (do NOT add
  `panic` to options — `WithLanguage`/`WithTranslation` ignore bad codes).
- `//////` section dividers; doc comment on every exported symbol; comments end
  with a period (godot is enabled).
- `Copy(src, target)` mutates and returns `target`, copying only NON-ZERO src
  fields; `LanguageMessageMap`/`Fields`/`Tags` are merged into fresh containers.
- Factory methods intentionally apply options twice (in `FormatError`/method
  body, then again in the inner package constructor) and depend on `Copy`'s
  "non-zero overwrites" merge for option precedence (e.g. a user `WithStatusCode`
  must win over a built-in default). Do NOT "dedupe" this without re-checking
  precedence — it is load-bearing.

## Gotchas
- `Error()` output is non-deterministic for >1 field (`sync.Map` range order);
  assert with `strings.Contains`, not equality.
- `example_test.go` asserts EXACT stdout — any message-format change breaks it.
- The template singleton (`languageprefixtemplate.go`) is package-global and
  persists across tests. In tests, register/use UNIQUE language codes (e.g.
  "xy", "qq") so you don't pollute other tests.
- `New`/`Factory` PANIC (recoverable) on invalid attributes: message `gte=3`,
  code `gte=2`, status `100..511`. They must NEVER call `os.Exit`/`log.Fatalf`.
- `From(*CustomError, …)` returns a modified COPY (never mutates the original);
  so `errors.Is(From(x,…), x)` is false for a `*CustomError` input.

## Build / test / lint
- `make test` / `make coverage` — `go test -race -cover` (passes; ~93%).
- `make lint` — golangci-lint with `.golangci.yml` (**v1 format**, `enable-all`).
  CI pins **golangci-lint v1.61.0** + **Go 1.23** (`.github/workflows/go.yml`).
  - A v2.x golangci-lint CANNOT read the v1 config (it errors). To reproduce CI
    locally, install v1.61.0 and run it under a go1.23 toolchain:
    `GOTOOLCHAIN=go1.23.x golangci-lint run -c .golangci.yml ./...`.
  - Running v1.61.0 against a **go1.24+** toolchain yields false
    `typecheck`/"undefined" errors (export-data mismatch) — not real issues.
  - `enable-all` is strict: with `go >= 1.22` in go.mod, `intrange`/`copyloopvar`
    turn on (use `for i := range n`); non-English template strings need
    `//nolint:misspell`.

## Dependency policy
- Keep deps at the HIGHEST version whose go.mod `go` directive is **≤ 1.23**
  (CI's Go). Absolute-latest `x/*`, `validator` require Go ≥ 1.24/1.25 and would
  break CI. Probe a version's requirement via
  `curl -s https://proxy.golang.org/<module>/@v/<ver>.mod | grep '^go '`.
- `go.mod` go directive is `1.23.0`; do NOT add a `toolchain` directive (a
  library shouldn't pin one).

## CI & releases
- The workflow runs ONLY on push to `main` and PRs targeting `main`. Pushing a
  feature branch does NOT trigger CI.
- Fresh clones may not fetch tags (`git fetch --tags`). Versioned via semver
  tags; latest tag `v1.2.9`, latest release `v1.2.7`. Removing an exported
  symbol is a breaking change → would need a `/v2` module path; prefer additive/
  deprecation over removal if a clean `v1.x` minor release is desired.
