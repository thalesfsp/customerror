// Copyright 2021 The customerror Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Tests for the v2.2.0 catalog features. Each block covers a happy path, a
// bad path, and edge cases:
//   - `Catalog.Set` stamps the validated, uppercased code on the stored entry
//     unless an explicit `WithErrorCode` was provided.
//   - `Catalog.MarshalJSON` produces a useful, deterministic JSON
//     representation of the catalog.

package customerror

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//////
// Feature: Catalog.Set stamps the code on the entry.
//////

func TestFeature_CatalogSet_CodeOnEntry(t *testing.T) {
	c := MustNewCatalog("myapp")

	// Happy path: Set then Get - the entry carries the code by default, is
	// matchable via IsErrorCode, prefixes Error(), and shows up in the JSON.
	code, err := c.Set("USER_NOT_FOUND", "user not found")
	require.NoError(t, err)
	assert.Equal(t, "USER_NOT_FOUND", code)

	entry, err := c.Get("USER_NOT_FOUND")
	require.NoError(t, err)
	assert.Equal(t, "USER_NOT_FOUND", entry.Code)
	assert.True(t, IsErrorCode(entry, "USER_NOT_FOUND"))
	assert.Equal(t, "USER_NOT_FOUND: user not found", entry.Error())

	payload, err := json.Marshal(entry)
	require.NoError(t, err)
	assert.JSONEq(t, `{"code":"USER_NOT_FOUND","message":"user not found"}`, string(payload))

	// Precedence: an explicit WithErrorCode must WIN over the catalog key.
	c.MustSet("KEY_CODE", "some message", WithErrorCode("OTHER"))

	entry = c.MustGet("KEY_CODE")
	assert.Equal(t, "OTHER", entry.Code)
	assert.True(t, IsErrorCode(entry, "OTHER"))
	assert.False(t, IsErrorCode(entry, "KEY_CODE"))
	assert.Equal(t, "OTHER: some message", entry.Error())

	// Edge: a lowercase input code is uppercased on the entry (NewErrorCode
	// stores codes uppercased).
	code, err = c.Set("lower_code", "some message")
	require.NoError(t, err)
	assert.Equal(t, "LOWER_CODE", code)
	assert.Equal(t, "LOWER_CODE", c.MustGet("lower_code").Code)

	// Edge: when Code == Message, Error() collapses to just the code (no
	// "CODE: CODE" duplication).
	c.MustSet("E1010", "E1010")
	assert.Equal(t, "E1010", c.MustGet("E1010").Error())

	// Edge: the stored entry is not mutated when a Get copy is modified (the
	// existing copy-on-Get guarantee still holds for the auto-set code).
	mutated := c.MustGet("USER_NOT_FOUND")
	mutated.Code = "MUTATED"
	mutated.SetMessage("mutated message")

	fresh := c.MustGet("USER_NOT_FOUND")
	assert.Equal(t, "USER_NOT_FOUND", fresh.Code)
	assert.Equal(t, "user not found", fresh.Message)

	// Bad path: an invalid code neither stores nor stamps anything.
	code, err = c.Set("BAD CODE!", "some message")
	require.Error(t, err)
	assert.Empty(t, code)
}

//////
// Feature: Catalog.MarshalJSON.
//////

func TestFeature_CatalogMarshalJSON(t *testing.T) {
	// Happy path: a catalog with 2+ entries marshals with its name, the codes
	// as keys, and each entry payload (message/code/statusCode) marshaled via
	// CustomError.MarshalJSON.
	c := MustNewCatalog("myapp")
	c.
		MustSet("USER_NOT_FOUND", "user not found", WithStatusCode(http.StatusNotFound)).
		MustSet("RATE_LIMITED", "too many requests", WithStatusCode(http.StatusTooManyRequests))

	payload, err := json.Marshal(c)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"name": "myapp",
		"custom_errors": {
			"RATE_LIMITED": {"code":"RATE_LIMITED","message":"too many requests","statusCode":429},
			"USER_NOT_FOUND": {"code":"USER_NOT_FOUND","message":"user not found","statusCode":404}
		}
	}`, string(payload))

	// Edge: deterministic output - marshaling twice is byte-identical
	// (encoding/json sorts map keys).
	payload2, err := json.Marshal(c)
	require.NoError(t, err)
	assert.Equal(t, payload, payload2)

	// Edge: an empty catalog marshals to an empty (but present) map.
	empty := MustNewCatalog("emptyapp")

	payload, err = json.Marshal(empty)
	require.NoError(t, err)
	assert.Equal(t, `{"name":"emptyapp","custom_errors":{}}`, string(payload))

	// Bad path (defensive): nil or foreign entries stored directly into the
	// map (bypassing Set) are skipped without panicking.
	poisoned := MustNewCatalog("poisonedapp")
	poisoned.MustSet("OK_CODE", "some message")
	poisoned.ErrorCodeErrorMap.Store(ErrorCode("NIL_TYPED"), (*CustomError)(nil))
	poisoned.ErrorCodeErrorMap.Store(ErrorCode("NIL_IFACE"), nil)
	poisoned.ErrorCodeErrorMap.Store("foreign_key", Factory("some message"))

	payload, err = json.Marshal(poisoned)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"name": "poisonedapp",
		"custom_errors": {
			"OK_CODE": {"code":"OK_CODE","message":"some message"}
		}
	}`, string(payload))
}
