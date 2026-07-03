// Copyright 2021 The customerror Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Coverage tests for the less-traveled paths: nil receivers, panic paths
// (Must* helpers), fallbacks, and defensive guards. Each block covers a happy
// path, a bad path, and edge cases.

package customerror

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//////
// Catalog Must* helpers.
//////

func TestCoverage_CatalogMustHelpers(t *testing.T) {
	// Happy path: MustSet returns the catalog for chaining, MustGet resolves.
	c := MustNewCatalog("myapp")
	assert.Same(t, c, c.MustSet("SOME_CODE", "some message"))
	assert.Equal(t, "some message", c.MustGet("SOME_CODE").Message)

	// Bad path: MustNewCatalog panics on an invalid (too short) name.
	assert.Panics(t, func() { MustNewCatalog("ab") })

	// Bad path: MustSet panics on an invalid error code.
	assert.Panics(t, func() { c.MustSet("BAD CODE!", "some message") })

	// Bad path: MustSet panics when the entry resolves to nil (ignore option).
	assert.Panics(t, func() { c.MustSet("IGNORED", "some message", WithIgnoreString("some")) })

	// Bad path: MustGet panics on a missing code.
	assert.Panics(t, func() { _ = c.MustGet("DOES_NOT_EXIST") })

	// Edge: Set with an invalid error code returns the error (no panic).
	code, err := c.Set("BAD CODE!", "some message")
	require.Error(t, err)
	assert.Empty(t, code)
	assert.ErrorIs(t, err, ErrErrorCodeInvalidCode)
}

//////
// Language helpers: MustAddNewLanguage, GetRoot, GetLanguageErrorTypeMap.
//////

func TestCoverage_LanguageHelpers(t *testing.T) {
	// Happy path: register a new language (unique code to avoid polluting the
	// package-global template singleton for other tests).
	assert.NotPanics(t, func() {
		MustAddNewLanguage("yo", NewErrorPrefixMap(
			"failed to %s", "invalid %s", "missing %s", "%s required", "%s not found",
		))
	})

	tpl, err := GetTemplate("yo", FailedTo.String())
	require.NoError(t, err)
	assert.Equal(t, "failed to %s", tpl)

	// Bad path: invalid language code panics.
	assert.Panics(t, func() {
		MustAddNewLanguage("not-a-lang", NewErrorPrefixMap("a %s", "b %s", "c %s", "d %s", "e %s"))
	})

	// Bad path: GetLanguageErrorTypeMap with an invalid code errors.
	_, err = GetLanguageErrorTypeMap("not-a-lang")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidLanguageCode)

	// Bad path: valid but unregistered language.
	_, err = GetLanguageErrorTypeMap("xv")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLanguageNotFound)

	// Edge: GetRoot.
	assert.Equal(t, "en", Language("en-US").GetRoot())
	assert.Equal(t, "en", Language("en").GetRoot())
	assert.Empty(t, Language("not-a-lang").GetRoot())
}

//////
// Nil receivers: FormatError and the instance factory methods must return nil
// instead of panicking.
//////

func TestCoverage_NilReceiverFactoryMethods(t *testing.T) {
	var cE *CustomError

	assert.Nil(t, cE.FormatError(FailedTo.String()))
	assert.NoError(t, cE.NewFailedToError())
	assert.NoError(t, cE.NewInvalidError())
	assert.NoError(t, cE.NewMissingError())
	assert.NoError(t, cE.NewRequiredError())
	assert.NoError(t, cE.NewHTTPError(http.StatusNotFound))
	assert.NoError(t, cE.New())
}

//////
// FormatError: root-language fallback ("pt-BR" -> "pt") and missing-template
// degradation.
//////

func TestCoverage_FormatError_RootFallback(t *testing.T) {
	// Happy path: "pt-BR" has no template map of its own; the built-in "pt"
	// root templates are used.
	err := Factory(
		"algo",
		WithTranslation("pt-BR", "algo"),
	).NewFailedToError(WithLanguage("pt-BR"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "falhou algo")

	// Edge: a language with a translation but no templates at all degrades to
	// the unprefixed message ("xv" is never registered).
	err = Factory(
		"algo",
		WithTranslation("xv", "unprefixed message"),
	).NewFailedToError(WithLanguage("xv"))

	require.Error(t, err)
	assert.Equal(t, "unprefixed message", err.(*CustomError).Message)
}

//////
// MarshalJSON: retried flag, status code, and empty-tags omission.
//////

func TestCoverage_MarshalJSON_Branches(t *testing.T) {
	cE := New(
		"some message",
		WithStatusCode(http.StatusServiceUnavailable),
		WithRetryable(true),
	).(*CustomError)

	cE.SetRetried(true)

	b, err := json.Marshal(cE)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &m))

	assert.Equal(t, "some message", m["message"])
	assert.Equal(t, true, m["retryable"])
	assert.Equal(t, true, m["retried"])
	assert.InEpsilon(t, float64(http.StatusServiceUnavailable), m["statusCode"], 0.001)

	// Edge: no code and no tags -> keys omitted.
	assert.NotContains(t, m, "code")
	assert.NotContains(t, m, "tags")
}

//////
// APIError: message-equals-status-text and message-equals-code branches.
//////

func TestCoverage_APIError_Branches(t *testing.T) {
	// Edge: message equals the HTTP status text -> no duplicated text.
	cE := New("Not Found", WithStatusCode(http.StatusNotFound)).(*CustomError)
	assert.Equal(t, "Not Found (404)", cE.APIError())

	// Edge: message equals the code -> code only, once.
	viaCode := newInternal(WithErrorCode("E1010"))
	assert.Equal(t, "E1010", viaCode.APIError())
	assert.Equal(t, "E1010", viaCode.Error())

	// Happy path: distinct message, code, and status.
	full := New("some message", WithErrorCode("E1010"), WithStatusCode(http.StatusNotFound)).(*CustomError)
	assert.Equal(t, "E1010: some message (404 - Not Found)", full.APIError())
}

//////
// Copy: ignore and retried flags propagate.
//////

func TestCoverage_Copy_Flags(t *testing.T) {
	src := &CustomError{Message: "some message", Retried: true, ignore: true}

	target := Copy(src, &CustomError{})

	assert.True(t, target.Retried)
	assert.True(t, target.ignore)
	assert.Equal(t, "some message", target.Message)
}
