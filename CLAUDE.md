# customerror — project guide for Claude

Go library providing rich, structured custom errors (`*CustomError`) with a code,
HTTP status code, tags, fields, i18n/translations, retryability, and an error
`Catalog`. Module path: `github.com/thalesfsp/customerror/v2` (v2 — importers use
the `/v2` suffix; the package name is still `customerror`).

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
- `Error()` field output is deterministic since v2.1.0: fields are sorted by
  key (`. Fields: a=1, b=2`); tags are sorted too (treeset).
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
- `make lint` — golangci-lint with `.golangci.yml` (**v2 format**, `default: all`).
  CI pins **golangci-lint v2.5.0** + **Go 1.25** (`.github/workflows/go.yml`,
  `golangci/golangci-lint-action@v8`). A modern (v2.x) golangci-lint runs the
  config directly — no version gymnastics needed.
  - The v2 migration disabled three linters that are new in v2 and clash with
    the repo's idioms, to preserve the prior baseline: `noinlineerr`,
    `funcorder`, `wsl_v5` (the old `wsl` stays). Re-enable deliberately if you
    want to adopt them (each needs codebase-wide changes).
  - `default: all` is strict: `intrange`/`copyloopvar` are on (use
    `for i := range n`); non-Latin template strings need a `gosmopolitan`
    exception (test files are already excluded) or `//nolint:gosmopolitan`.

## Dependency policy
- Tracks the **latest** releases (go.mod `go` directive is `1.25.0`, matching CI).
  When bumping, confirm a candidate's required Go via
  `curl -s https://proxy.golang.org/<module>/@v/<ver>.mod | grep '^go '` and keep
  it ≤ the CI Go. Do NOT add a `toolchain` directive (a library shouldn't pin one).

## CI & releases
- The workflow runs ONLY on push to `main` and PRs targeting `main`. Pushing a
  feature branch does NOT trigger CI.
- Fresh clones may not fetch tags (`git fetch --tags`). Versioned via semver
  tags. This module is **v2** (`/v2` path); the v1 line stopped at `v1.2.9`.
  A breaking change now requires a `/v3` path; prefer additive/deprecation
  within `v2.x`.
