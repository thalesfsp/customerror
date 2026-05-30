// Copyright 2021 The customerror Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Regression tests for the bug fixes. Each test block covers a happy path, a
// bad/buggy path (the behavior that was broken before the fix), and edge cases.

package customerror

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//////
// #1 - New must NOT terminate the process (no log.Fatalf/os.Exit). Invalid
// input panics (recoverable) instead.
//////

func TestFix_New_InvalidInputPanicsButDoesNotExit(t *testing.T) {
	// Bad path: invalid attributes must panic (recoverable), NOT call os.Exit.
	mustPanic := func(message string) {
		defer func() {
			r := recover()
			require.NotNil(t, r, "expected a (recoverable) panic for message %q", message)
			assert.Contains(t, fmt.Sprintf("%v", r), "Invalid custom error")
		}()

		_ = New(message)
	}

	mustPanic("")   // empty message
	mustPanic("ab") // 2 chars, below the gte=3 minimum

	// Reaching this point proves the process was NOT terminated by the panics
	// above (a log.Fatalf/os.Exit would have killed the test binary).

	// Happy path + edge: exactly 3 chars (the boundary) is valid.
	assert.NotPanics(t, func() {
		assert.NotNil(t, New("abc"))
		assert.NotNil(t, New("a valid message"))
	})
}

//////
// #2 - The NewHTTPError method must not mutate its receiver.
//////

func TestFix_NewHTTPError_DoesNotMutateReceiver(t *testing.T) {
	base := Factory("something")

	// Happy path.
	first, ok := To(base.NewHTTPError(http.StatusNotFound))
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, first.StatusCode)
	assert.Equal(t, "not found", first.Message)

	// The receiver must remain untouched.
	assert.Equal(t, 0, base.StatusCode, "receiver StatusCode must not be mutated")

	// Bad path (the bug): reusing the same factory must NOT make the first
	// status code "stick".
	second, ok := To(base.NewHTTPError(http.StatusInternalServerError))
	require.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, second.StatusCode)
	assert.Equal(t, 0, base.StatusCode)

	// Edge: a receiver with a preset status code keeps it (receiver wins), and
	// is still not mutated.
	preset := Factory("teapot", WithStatusCode(http.StatusTeapot))
	got, ok := To(preset.NewHTTPError(http.StatusNotFound))
	require.True(t, ok)
	assert.Equal(t, http.StatusTeapot, got.StatusCode)
	assert.Equal(t, http.StatusTeapot, preset.StatusCode)

	// Edge: nil receiver returns nil.
	var nilCE *CustomError
	assert.Nil(t, nilCE.NewHTTPError(http.StatusNotFound))
}

//////
// #3 - MarshalJSON: user fields must not clobber structural keys.
//////

func TestFix_MarshalJSON_FieldsDoNotClobberReservedKeys(t *testing.T) {
	// Happy path: ordinary fields are emitted at the top level.
	okFields := New("real message", WithErrorCode("E1010"), WithField("custom", "value")).(*CustomError)
	b, err := json.Marshal(okFields)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Equal(t, "value", m["custom"])
	assert.Equal(t, "E1010", m["code"])

	// Bad path (the bug): a field named like a reserved key must NOT override
	// the structural value.
	hijack := New(
		"real message",
		WithErrorCode("E1010"),
		WithStatusCode(http.StatusBadRequest),
		WithField("code", "HIJACKED"),
		WithField("message", "HIJACKED"),
		WithField("statusCode", 999),
	).(*CustomError)

	b, err = json.Marshal(hijack)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &m))

	assert.Equal(t, "E1010", m["code"], "field must not clobber code")
	assert.Equal(t, "real message", m["message"], "field must not clobber message")
	assert.Equal(t, float64(http.StatusBadRequest), m["statusCode"], "field must not clobber statusCode")

	// Edge: a "statusCode" field with no real status code is dropped (not
	// emitted as the field value either).
	noStatus := New("msg here", WithField("statusCode", 12345)).(*CustomError)
	b, err = json.Marshal(noStatus)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "12345")

	// Edge: empty key and nil value are skipped.
	weird := New("msg here", WithField("", "x"), WithField("nilval", nil), WithField("good", 1)).(*CustomError)
	b, err = json.Marshal(weird)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &m))
	_, hasEmpty := m[""]
	assert.False(t, hasEmpty)
	_, hasNil := m["nilval"]
	assert.False(t, hasNil)
	assert.Equal(t, float64(1), m["good"])
}

//////
// #4 - WithFields must merge (not wipe existing fields).
//////

func TestFix_WithFields_Merges(t *testing.T) {
	// Happy path: all keys present.
	e := New("message", WithFields(map[string]interface{}{"a": 1, "b": 2})).(*CustomError)
	assertField(t, e, "a", 1)
	assertField(t, e, "b", 2)

	// Bad path (the bug): WithField then WithFields must keep both.
	e = New("message",
		WithField("a", 1),
		WithFields(map[string]interface{}{"b": 2}),
	).(*CustomError)
	assertField(t, e, "a", 1)
	assertField(t, e, "b", 2)

	// Edge: an empty map preserves existing fields (and adds no dangling state).
	e = New("message",
		WithField("a", 1),
		WithFields(map[string]interface{}{}),
	).(*CustomError)
	assertField(t, e, "a", 1)

	// Edge: WithFields twice merges; same key is overwritten with the latest.
	e = New("message",
		WithFields(map[string]interface{}{"a": 1}),
		WithFields(map[string]interface{}{"a": 99, "b": 2}),
	).(*CustomError)
	assertField(t, e, "a", 99)
	assertField(t, e, "b", 2)
}

func assertField(t *testing.T, e *CustomError, key string, want interface{}) {
	t.Helper()

	require.NotNil(t, e.Fields)
	got, ok := e.Fields.Load(key)
	assert.True(t, ok, "expected field %q to be present", key)
	assert.Equal(t, want, got)
}

//////
// #5 - Catalog.Get/MustGet: apply opts and return a copy (not the shared
// pointer).
//////

func TestFix_CatalogGet_AppliesOptsAndReturnsCopy(t *testing.T) {
	c := MustNewCatalog("myapp")
	c.MustSet("E1010", "original message")

	// Happy path.
	got, err := c.Get("E1010")
	require.NoError(t, err)
	assert.Equal(t, "original message", got.Message)

	// Bad path (the bug 1): opts passed to Get/MustGet must be applied.
	withOpt := c.MustGet("E1010", WithStatusCode(http.StatusTeapot))
	assert.Equal(t, http.StatusTeapot, withOpt.StatusCode)

	// The stored entry must be unaffected by the applied opt.
	stored, err := c.Get("E1010")
	require.NoError(t, err)
	assert.NotEqual(t, http.StatusTeapot, stored.StatusCode)

	// Bad path (the bug 2): mutating the returned error must not corrupt the
	// catalog, and two Gets must return distinct pointers.
	got1, _ := c.Get("E1010")
	got2, _ := c.Get("E1010")
	assert.True(t, got1 != got2, "Get must return distinct copies")

	got1.SetMessage("MUTATED")
	fresh, _ := c.Get("E1010")
	assert.Equal(t, "original message", fresh.Message, "catalog entry must not be mutated")

	// Bad path: not found / invalid code.
	_, err = c.Get("DOES_NOT_EXIST")
	assert.ErrorIs(t, err, ErrCatalogErrorNotFound)

	_, err = c.Get("invalid code!")
	assert.Error(t, err)

	// Edge: MustGet panics on a missing code.
	assert.Panics(t, func() { _ = c.MustGet("DOES_NOT_EXIST") })
}

//////
// #6 - ErrorCode regex must actually validate (anchored).
//////

func TestFix_ErrorCodeRegex_Anchored(t *testing.T) {
	// Happy path: legitimate codes are accepted.
	for _, code := range []string{
		"E1010", "INVALID_REQUEST_BODY", "ERR_INVALID_HARD_DRIVE_PATH",
		"CE_ERR_INVALID_LANG_CODE", "ERR_A1_B2", "E12345678", "AbCd123", "X", "_",
	} {
		_, err := NewErrorCode(code)
		assert.NoError(t, err, "expected %q to be valid", code)
	}

	// Bad path (the bug): junk with spaces/punctuation must be rejected.
	for _, code := range []string{
		"hello world", "!!!abc!!!", "@@@", "ERR-123", "", "a b", "x.y",
	} {
		_, err := NewErrorCode(code)
		assert.ErrorIs(t, err, ErrErrorCodeInvalidCode, "expected %q to be rejected", code)
	}
}

//////
// #7 - Language regex must be fully anchored ("default" alternative).
//////

func TestFix_LanguageRegex_Anchored(t *testing.T) {
	// Happy path.
	for _, lang := range []string{"en", "en-US", "pt-BR", "default"} {
		_, err := NewLanguage(lang)
		assert.NoError(t, err, "expected %q to be valid", lang)
	}

	// Bad path (the bug): partial "default" matches must be rejected, along
	// with malformed codes.
	for _, lang := range []string{"mydefault", "defaultx", "xdefault", "EN", "e", "eng", "en-us", "en-USA", ""} {
		_, err := NewLanguage(lang)
		assert.ErrorIs(t, err, ErrInvalidLanguageCode, "expected %q to be rejected", lang)
	}

	// Edge: GetRoot keeps working with the (preserved) single capture group.
	assert.Equal(t, "en", Language("en-US").GetRoot())
	assert.Equal(t, "pt", Language("pt-BR").GetRoot())
	assert.Equal(t, "en", Language("en").GetRoot())
	assert.Equal(t, "default", Language("default").GetRoot())
}

//////
// #8 - From must return a copy, never mutate the original.
//////

func TestFix_From_DoesNotMutateOriginal(t *testing.T) {
	sentinel := New("sentinel error", WithErrorCode("E1010")).(*CustomError)

	// Happy path: the returned error has the new options applied.
	modified := From(sentinel, WithErrorCode("E9999")).(*CustomError)
	assert.Equal(t, "E9999", modified.Code)

	// Bad path (the bug): the original sentinel must be unchanged.
	assert.Equal(t, "E1010", sentinel.Code, "From must not mutate the original error")

	// Edge: From on a non-CustomError wraps it, applies opts, and stays
	// chain-compatible.
	plain := errors.New("plain error")
	wrapped := From(plain, WithErrorCode("E1234")).(*CustomError)
	assert.Equal(t, "E1234", wrapped.Code)
	assert.ErrorIs(t, wrapped, plain)

	// Edge: From with no opts returns an independent copy.
	cp := From(sentinel).(*CustomError)
	assert.True(t, cp != sentinel, "From must return a distinct copy")
	cp.SetMessage("changed")
	assert.Equal(t, "sentinel error", sentinel.Message)
}

//////
// #9 - No dangling ". Tags:" / ". Fields:" for empty (non-nil) sets/maps.
//////

func TestFix_EmptyTagsFields_NoDanglingOutput(t *testing.T) {
	// Bad path (the bug): empty (but non-nil) tags and fields must not render.
	e := New("boom", WithTag(), WithFields(map[string]interface{}{})).(*CustomError)
	assert.Equal(t, "boom", e.Error())
	assert.Equal(t, "boom", e.APIError())

	// Edge: only-empty-tags and only-empty-fields independently produce nothing.
	assert.Equal(t, "boom", New("boom", WithTag()).(*CustomError).Error())
	assert.Equal(t, "boom", New("boom", WithFields(map[string]interface{}{})).(*CustomError).Error())

	// Edge: MarshalJSON must omit an empty "tags" key.
	b, err := json.Marshal(e)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "tags")

	// Happy path: non-empty tags and fields still render.
	full := New("boom", WithTag("t1", "t2"), WithField("k", "v")).(*CustomError)
	assert.Contains(t, full.Error(), ". Tags: t1, t2")
	assert.Contains(t, full.Error(), ". Fields:")
	assert.Contains(t, full.Error(), "k=v")
}

//////
// #10 - Factory methods apply options correctly despite internal re-application
// (idempotency regression).
//////

func TestFix_FactoryMethods_OptionPrecedenceAndNoDuplication(t *testing.T) {
	base := Factory("disk")

	// A WithStatusCode override must win over the built-in default (500).
	failed, ok := To(base.NewFailedToError(WithStatusCode(http.StatusTeapot)))
	require.True(t, ok)
	assert.Equal(t, http.StatusTeapot, failed.StatusCode)
	assert.Equal(t, "failed to disk", failed.Message)

	// Default status code is still applied when not overridden.
	failedDefault, ok := To(base.NewFailedToError())
	require.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, failedDefault.StatusCode)

	// Tags/fields are not duplicated.
	tagged, ok := To(base.NewInvalidError(WithTag("t1"), WithField("k", "v")))
	require.True(t, ok)
	assert.Equal(t, 1, tagged.Tags.Size())
	assertField(t, tagged, "k", "v")
}

//////
// #11 - Options and FormatError must not panic on bad input.
//////

func TestFix_NoPanicOnInvalidLanguage(t *testing.T) {
	// Bad path (the bug): an invalid language code must not panic.
	assert.NotPanics(t, func() {
		e := New("base message", WithLanguage("not-a-lang")).(*CustomError)
		assert.Equal(t, "base message", e.Message) // default kept
	})

	assert.NotPanics(t, func() {
		_ = New("base message", WithTranslation("BAD_CODE", "x"))
	})

	// Edge: a valid code that is not present in the map is ignored (default).
	assert.NotPanics(t, func() {
		e := New("base message", WithLanguage("fr")).(*CustomError)
		assert.Equal(t, "base message", e.Message)
	})

	// Happy path: a valid, present translation is used.
	e := New("base message",
		WithTranslation("fr", "message français"),
		WithLanguage("fr"),
	).(*CustomError)
	assert.Equal(t, "message français", e.Message)
}

func TestFix_FormatError_NoTemplateDegradesGracefully(t *testing.T) {
	// Register a language whose template map is intentionally incomplete (only
	// has the "failed to" template, not "invalid").
	partial := &sync.Map{}
	partial.Store(FailedTo, "qq-failed %s")
	require.NoError(t, AddNewLanguage("qq", partial))

	cE := Factory("thing", WithTranslation("qq", "translated thing"))

	// Bad path (the bug): a missing template must NOT panic; it degrades to the
	// unprefixed (translated) message.
	var result error
	assert.NotPanics(t, func() {
		result = cE.NewInvalidError(WithLanguage("qq"))
	})
	assert.Equal(t, "translated thing", result.Error())
}

//////
// #12 - Chinese must use the correct ISO 639-1 code ("zh").
//////

func TestFix_ChineseLanguageCode(t *testing.T) {
	// Happy path / correctness.
	assert.Equal(t, "zh", string(Chinese))
	assert.Contains(t, BuiltInLanguages, "zh")
	assert.NotContains(t, BuiltInLanguages, "ch")

	tmpl, err := GetTemplate("zh", string(FailedTo))
	require.NoError(t, err)
	assert.Equal(t, "无法 %s", tmpl)

	// Edge: the old (wrong) "ch" code is no longer a registered language.
	_, err = GetTemplate("ch", string(FailedTo))
	assert.ErrorIs(t, err, ErrLanguageNotFound)
}

//////
// #13 - Italian "failed to" template spelling.
//////

func TestFix_ItalianFailedToSpelling(t *testing.T) {
	tmpl, err := GetTemplate("it", string(FailedTo))
	require.NoError(t, err)
	assert.Equal(t, "impossibile %s", tmpl) //nolint:misspell // correct Italian, not an English typo
}

//////
// #14 - The dead LanguageErrorTypeMap field is gone (not serialized).
//////

func TestFix_NoLanguageErrorTypeMapInJSON(t *testing.T) {
	b, err := json.Marshal(New("a message", WithErrorCode("E1010")).(*CustomError))
	require.NoError(t, err)
	assert.NotContains(t, string(b), "languageErrorTypeMap")
}

//////
// #15 - The shared validator instance is safe under concurrent use.
//////

func TestFix_ConcurrentNewIsRaceFree(t *testing.T) {
	const n = 50

	results := make(chan error, n)

	for i := 0; i < n; i++ {
		go func(i int) {
			results <- New(fmt.Sprintf("message %d", i), WithErrorCode("E1010"), WithStatusCode(http.StatusBadRequest))
		}(i)
	}

	for i := 0; i < n; i++ {
		assert.NotNil(t, <-results)
	}
}

//////
// #16 - Factory honors WithIgnoreFunc (and the removed dead nil-checks don't
// change behavior).
//////

func TestFix_FactoryAndNewHappyAndIgnore(t *testing.T) {
	// Happy path.
	assert.NotNil(t, Factory("a valid message"))

	// Ignore path: Factory returns nil when ignored.
	assert.Nil(t, Factory("ignore me", WithIgnoreString("ignore")))

	// Ignore path: New returns nil when ignored.
	assert.Nil(t, New("ignore me", WithIgnoreString("ignore")))
}

//////
// #17 - AddNewLanguage can update an existing language (Store, not LoadOrStore).
//////

func TestFix_AddNewLanguage_UpdatesExisting(t *testing.T) {
	// Happy path: register a brand-new language.
	require.NoError(t, AddNewLanguage("xy", NewErrorPrefixMap(
		"xy-failed %s", "xy-invalid %s", "xy-missing %s", "xy-required %s", "xy-notfound %s",
	)))

	tmpl, err := GetTemplate("xy", string(FailedTo))
	require.NoError(t, err)
	assert.Equal(t, "xy-failed %s", tmpl)

	// Bad path: an invalid language code returns an error (and does not panic).
	assert.Error(t, AddNewLanguage("INVALID", NewErrorPrefixMap("a", "b", "c", "d", "e")))

	// Edge (the bug): re-adding the same language must overwrite, not no-op.
	require.NoError(t, AddNewLanguage("xy", NewErrorPrefixMap(
		"xy2-failed %s", "xy2-invalid %s", "xy2-missing %s", "xy2-required %s", "xy2-notfound %s",
	)))

	tmpl, err = GetTemplate("xy", string(FailedTo))
	require.NoError(t, err)
	assert.Equal(t, "xy2-failed %s", tmpl)
}

//////
// Catalog name validation (gte=3, consistent with the rest).
//////

func TestFix_CatalogName_Validation(t *testing.T) {
	// Happy path.
	_, err := NewCatalog("myapp")
	assert.NoError(t, err)

	// Edge: exactly 3 chars is now valid (boundary).
	_, err = NewCatalog("abc")
	assert.NoError(t, err)

	// Bad path: empty and too-short names are rejected.
	_, err = NewCatalog("")
	assert.ErrorIs(t, err, ErrCatalogInvalidName)

	_, err = NewCatalog("ab")
	assert.ErrorIs(t, err, ErrCatalogInvalidName)
}

//////
// prependOptions must not mutate the caller's slice (no append aliasing).
//////

func TestFix_PrependOptions_NoAliasing(t *testing.T) {
	var order []string

	mk := func(name string) Option {
		return func(*CustomError) { order = append(order, name) }
	}

	// Happy path: item is first, originals follow in order.
	source := []Option{mk("b"), mk("c")}
	result := prependOptions(source, mk("a"))

	ce := &CustomError{}
	for _, o := range result {
		o(ce)
	}
	assert.Equal(t, []string{"a", "b", "c"}, order)

	// Edge (the bug): even with spare capacity, the caller's slice is intact.
	order = nil
	src := make([]Option, 2, 8)
	src[0] = mk("b")
	src[1] = mk("c")
	_ = prependOptions(src, mk("a"))

	for _, o := range src {
		o(ce)
	}
	assert.Equal(t, []string{"b", "c"}, order, "caller's slice must be unchanged")

	// Edge: empty source.
	order = nil
	for _, o := range prependOptions(nil, mk("a")) {
		o(ce)
	}
	assert.Equal(t, []string{"a"}, order)
}

//////
// Is must not panic when comparing non-comparable wrapped errors.
//////

type uncomparableError struct{ _ []int }

func (uncomparableError) Error() string { return "uncomparable" }

func TestFix_Is_NonComparableNoPanic(t *testing.T) {
	// Bad path (the bug): comparing non-comparable error values must not panic.
	cE := New("wrapper", WithError(uncomparableError{})).(*CustomError)

	assert.NotPanics(t, func() {
		assert.False(t, errors.Is(cE, uncomparableError{}))
	})

	// Happy path: comparable wrapped errors still match correctly.
	target := errors.New("target")
	wrapper := New("wrapper", WithError(target)).(*CustomError)
	assert.True(t, errors.Is(wrapper, target))
	assert.False(t, errors.Is(wrapper, errors.New("other")))
}

//////
// #20 - Wrap preserves the identity of every wrapped error.
//////

func TestFix_Wrap_PreservesIdentity(t *testing.T) {
	primary := New("primary", WithErrorCode("E1010"))
	extra1 := errors.New("extra one")
	extra2 := errors.New("extra two")

	wrapped := Wrap(primary, extra1, extra2)

	// Happy path: the message format is preserved.
	assert.Equal(t, "E1010: primary. Wrapped Error(s): extra one. extra two", wrapped.Error())

	// The fix: errors.Is matches the primary AND every extra error.
	assert.ErrorIs(t, wrapped, primary)
	assert.ErrorIs(t, wrapped, extra1)
	assert.ErrorIs(t, wrapped, extra2)

	// errors.As can still reach the wrapped CustomError.
	var cE *CustomError
	assert.True(t, errors.As(wrapped, &cE))
	assert.Equal(t, "E1010", cE.Code)

	// Edge: no extra errors.
	only := Wrap(primary)
	assert.ErrorIs(t, only, primary)

	// Edge: nil errors are skipped (message and identity).
	withNil := Wrap(primary, nil, extra1, nil)
	assert.Equal(t, "E1010: primary. Wrapped Error(s): extra one", withNil.Error())
	assert.ErrorIs(t, withNil, extra1)
}

//////
// #21 - Instance factory methods must not panic (nil type assertion) when an
// ignore option fires; they return nil instead.
//////

func TestFix_FactoryMethods_IgnoreReturnsNilNoPanic(t *testing.T) {
	base := Factory("create file")

	cases := []struct {
		name string
		call func() error
	}{
		{"NewFailedToError", func() error { return base.NewFailedToError(WithIgnoreString("failed")) }},
		{"NewInvalidError", func() error { return base.NewInvalidError(WithIgnoreString("invalid")) }},
		{"NewMissingError", func() error { return base.NewMissingError(WithIgnoreString("missing")) }},
		{"NewRequiredError", func() error { return base.NewRequiredError(WithIgnoreString("required")) }},
		{"NewHTTPError", func() error { return base.NewHTTPError(http.StatusNotFound, WithIgnoreString("not found")) }},
		{"New", func() error { return base.New(WithIgnoreString("create")) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got error
			assert.NotPanics(t, func() { got = tc.call() })
			assert.Nil(t, got, "ignored error must be nil")
		})
	}

	// Happy path: without an ignore match, a proper error is returned.
	failed := base.NewFailedToError()
	require.NotNil(t, failed)
	assert.Contains(t, failed.Error(), "failed to create file")
}

// Guard against accidental cross-test pollution of the shared comparator: the
// Set stringer must stay deterministic.
func TestFix_Sanity_SetStringDeterministic(t *testing.T) {
	e := New("msg", WithTag("b", "a", "c")).(*CustomError)
	assert.Equal(t, "a, b, c", strings.TrimSpace(e.Tags.String()))
}
