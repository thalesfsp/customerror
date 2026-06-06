# customerror — Improvement Backlog (post‑v2.0.0)

Follow‑up suggestions captured after the v2.0.0 bug‑fix/hardening release.
Pick by ID. File references point at the current `main`.

---

## 0. Cleanup (still pending — blocked in the cloud sandbox, 1‑click locally)

| ID | Task | Notes |
|----|------|-------|
| X1 | Delete merged branch `claude/vibrant-sagan-HNwCr` | already merged into `main` via PR #4 |
| X2 | Delete orphaned branches `dependabot/go_modules/golang.org/x/crypto-0.17.0` and `…/x/net-0.17.0` | their PRs #2/#3 are already **closed**; branches are obsolete |

```bash
git push origin --delete claude/vibrant-sagan-HNwCr
git push origin --delete dependabot/go_modules/golang.org/x/crypto-0.17.0
git push origin --delete dependabot/go_modules/golang.org/x/net-0.17.0
# or just enable "Automatically delete head branches" in repo settings
```

---

## ErrorCatalog improvements (`catalog.go`)

### C1 — Introspection: `List()` / `Each()` / `Has()`
**What:** Add read APIs to enumerate the catalog.
**Why:** Today it is write‑only (`Set`/`Get`); you can't list or audit registered codes.
**Sketch:**
```go
func (c *Catalog) Has(code string) bool
func (c *Catalog) List() []string                       // sorted error codes
func (c *Catalog) Each(fn func(code string, cE *CustomError) bool) // ranges (copies)
```

### C2 — `GenerateDocs()` → Markdown/JSON error reference
**What:** Emit a reference doc (code, message, status, tags, translations) for the whole catalog.
**Why:** Delivers the project's stated goal ("a catalog of errors … improves customer service"). Great for support/runbooks.
**Sketch:**
```go
func (c *Catalog) MarshalJSON() ([]byte, error)
func (c *Catalog) ToMarkdown() string   // table: Code | Status | Message | Tags | Langs
```

### C3 — Duplicate‑code guard on `Set`
**What:** Return an error (or `SetUnique`) when a code is registered twice instead of silently overwriting.
**Why:** Silent overwrite hides bugs as catalogs grow.
**Sketch:** `Set` returns `ErrCatalogDuplicateCode` if `Has(code)` (keep `MustSet` for intentional overrides).

### C4 — Load/Save catalog as data (YAML/JSON)
**What:** Define errors declaratively and load them; dump them back.
**Why:** Decouples error definitions from code; share one source of truth across services/languages.
**Sketch:**
```go
func LoadCatalog(r io.Reader) (*Catalog, error)   // yaml/json: code -> {message,status,tags,translations}
func (c *Catalog) Save(w io.Writer) error
```

### C5 — `Validate()` translation completeness
**What:** Verify every entry has a translation for each registered/required language.
**Why:** Catch missing‑locale gaps in CI before they reach users.
**Sketch:** `func (c *Catalog) Validate(required ...string) []error` (one error per code/lang gap).

### C6 — Per‑entry severity / category
**What:** Optional `Severity` (info/warn/error/critical) and `Category` fields + filters.
**Why:** Enables triage, alert routing, and log‑level mapping from the error itself.
**Sketch:** new options `WithSeverity`, `WithCategory`; `Catalog.FilterBySeverity(...)`.

---

## i18n improvements (`language.go`, `languageprefixtemplate.go`, `options.go`)

### I1 — Named / multiple placeholders (ICU‑style)
**What:** Support messages with >1 variable and reordering, beyond the single `%s` in `FormatError`.
**Why:** Real messages need multiple args, and word order differs per language.
**Sketch:** template `"{drive} is invalid ({code})"` + `WithVars(map[string]any)`; render via `text/template` or an ICU lib.

### I2 — Validate language codes with `golang.org/x/text/language` (BCP‑47)
**What:** Replace the hand‑rolled regex with `language.Parse`.
**Why:** The regex let real bugs through (the `ch`→`zh` mistake; junk like `xdefault` before the v2 fix). x/text is canonical and gives proper matching/fallback.
**Sketch:** `NewLanguage` wraps `language.Parse`; use `language.NewMatcher` for fallback selection.

### I3 — CLDR pluralization via `golang.org/x/text/message`
**What:** Correct singular/plural (and gender where supported) per locale.
**Why:** `%s` templates can't express "1 file" vs "2 files" across languages.
**Sketch:** back the templates with `message.NewPrinter(tag)` + `catalog` plural rules.

### I4 — Default language + `Accept-Language` helper
**What:** Configurable global default and a parser for HTTP `Accept-Language`.
**Why:** Completes the fallback chain (requested → root → default → English) and eases web use.
**Sketch:**
```go
func SetDefaultLanguage(lang string) error
func LanguageFromAcceptHeader(h string) string   // best match among registered langs
```

### I5 — `context.Context` language propagation
**What:** Carry the locale in context and let errors resolve from it.
**Why:** Avoids threading a `WithLanguage` arg through every call in request handlers.
**Sketch:** `ContextWithLanguage(ctx, lang)`, `LanguageFromContext(ctx)`, `cE.NewInvalidErrorCtx(ctx)`.

### I6 — Missing‑translation detector (test/CI helper)
**What:** A helper that asserts every catalog entry and every built‑in error type has all configured languages.
**Why:** Make i18n coverage enforceable in CI.
**Sketch:** `func CheckTranslations(c *Catalog, langs ...string) []error` + an example test.

---

## Suggested order

1. **Quick wins / correctness:** I2, C3, C5, I6
2. **High user value:** C2, I1, I4
3. **Larger / architectural:** C4, I3, I5, C1, C6

> Note: I1/I3/I4/I5 may benefit from adding `golang.org/x/text` as a direct dependency.
> Any change touching message formatting will break `example_test.go` (it asserts exact stdout) — update those Examples in lockstep.
