// Copyright 2021 The customerror Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// End-to-end tests: exercise the library the way an application would, from
// catalog setup through an HTTP round trip to the JSON the client receives,
// plus the i18n and retry journeys. Each scenario covers a happy path, a bad
// path, and edge cases.

package customerror

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_HTTPAPIErrorFlow simulates a service that registers its errors in a
// catalog at startup, and returns them - enriched per-request - as JSON over
// HTTP.
func TestE2E_HTTPAPIErrorFlow(t *testing.T) {
	// Application startup: build the error catalog. NOTE: since v2.2.0 Set
	// also stamps the code on the entry itself, so it travels with the error
	// (message prefix, JSON, IsErrorCode) without an explicit WithErrorCode.
	catalog := MustNewCatalog("userservice")
	catalog.
		MustSet("USER_NOT_FOUND", "user",
			WithStatusCode(http.StatusNotFound),
			WithTag("user", "lookup"),
		).
		MustSet("RATE_LIMITED", "too many requests",
			WithStatusCode(http.StatusTooManyRequests),
			WithRetryable(true),
		)

	// The HTTP handler resolves catalog errors per request.
	mux := http.NewServeMux()
	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/users/"):]

		if id == "42" {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"id":"42"}`)

			return
		}

		cE := catalog.MustGet("USER_NOT_FOUND", WithField("id", id))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(cE.StatusCode)

		// NOTE: assert (not require) - this runs on the server goroutine, where
		// FailNow is not allowed.
		payload, err := json.Marshal(cE)
		assert.NoError(t, err)
		_, _ = w.Write(payload)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	get := func(path string) *http.Response {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+path, nil)
		require.NoError(t, err)

		resp, err := srv.Client().Do(req)
		require.NoError(t, err)

		return resp
	}

	// Happy path: existing user.
	resp := get("/users/42")

	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Bad path: missing user - the client receives the structured error.
	resp2 := get("/users/99")

	defer func() { _ = resp2.Body.Close() }()

	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)

	body, err := io.ReadAll(resp2.Body)
	require.NoError(t, err)

	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &got))

	assert.Equal(t, "user", got["message"])
	assert.Equal(t, "USER_NOT_FOUND", got["code"])
	assert.Equal(t, "99", got["id"])
	assert.InEpsilon(t, float64(http.StatusNotFound), got["statusCode"], 0.001)

	// Edge: per-request enrichment never leaks back into the catalog.
	pristine := catalog.MustGet("USER_NOT_FOUND")
	if pristine.Fields != nil {
		_, leaked := pristine.Fields.Load("id")
		assert.False(t, leaked, "per-request field must not pollute the catalog entry")
	}

	// The retrieved error is matchable by its code.
	assert.True(t, IsErrorCode(pristine, "USER_NOT_FOUND"))
}

// TestE2E_RetryFlow simulates a client retrying an operation based on
// IsRetryable, flipping SetRetried on a per-attempt copy, with the final
// error wrapped and matched through the chain.
func TestE2E_RetryFlow(t *testing.T) {
	// A shared sentinel, as an application would declare at package level.
	errUpstream := errors.New("connection reset")
	rateLimited := Factory(
		"too many requests",
		WithStatusCode(http.StatusTooManyRequests),
		WithErrorCode("RATE_LIMITED"),
		WithRetryable(true),
	)

	attempts := 0
	operation := func() error {
		attempts++
		if attempts < 3 {
			return rateLimited.New(WithError(errUpstream))
		}

		return nil
	}

	// Happy path: retry until success.
	var last error

	for range 5 {
		last = operation()
		if last == nil {
			break
		}

		require.True(t, IsRetryable(last), "should be marked retryable")

		cE, ok := To(last)
		require.True(t, ok)
		cE.SetRetried(true)

		// The shared factory is never mutated by per-attempt changes.
		assert.False(t, rateLimited.Retried)
	}

	assert.NoError(t, last)
	assert.Equal(t, 3, attempts)

	// Bad path: a non-retryable error stops the loop logic.
	fatal := New("some fatal problem")
	assert.False(t, IsRetryable(fatal))

	// Edge: matching through a wrapped chain - Wrap + fmt.Errorf.
	failure := rateLimited.New(WithError(errUpstream))
	wrapped := fmt.Errorf("job failed: %w", Wrap(failure, errors.New("cleanup also failed")))

	assert.True(t, IsErrorCode(wrapped, "RATE_LIMITED"))
	assert.True(t, IsHTTPStatus(wrapped, http.StatusTooManyRequests))
	assert.True(t, IsRetryable(wrapped))
	assert.ErrorIs(t, wrapped, errUpstream)
}

// TestE2E_I18nFlow exercises the full translation journey: registering a
// language, translating catalog errors, and falling back gracefully.
func TestE2E_I18nFlow(t *testing.T) {
	// Startup: register a custom language (unique code "kw" to avoid polluting
	// the package-global template singleton for other tests).
	MustAddNewLanguage("kw", NewErrorPrefixMap(
		"fallu a %s",
		"%s invalidu",
		"falta %s",
		"%s riquisitu",
		"%s nun s'atopa",
	))

	base := Factory(
		"la conexón",
		WithTranslation("kw", "la conexón"),
		WithTranslation("en", "the connection"),
	)

	// Happy path: translated + prefixed message.
	err := base.NewFailedToError(WithLanguage("kw"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fallu a la conexón")

	// Happy path: built-in language via the same flow.
	err = base.NewFailedToError(WithLanguage("en"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to the connection")

	// Bad path: unknown language falls back to the default message.
	err = base.NewFailedToError(WithLanguage("xv"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to la conexón")

	// Edge: invalid language code is ignored gracefully (no panic, default
	// message).
	err = base.NewFailedToError(WithLanguage("not-a-lang"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to la conexón")
}
