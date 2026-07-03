// Copyright 2021 The customerror Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Concurrency tests for the Set type (the Tags container). Each block covers
// a happy path, a bad path, and edge cases. These tests are meant to be run
// with the race detector enabled (`go test -race`).

package customerror

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//////
// Happy path: concurrent WithTag on the SAME *CustomError while other
// goroutines read it (Error, json.Marshal). No data race, and afterwards all
// tags are present and sorted.
//////

func TestSetConcurrency_WithTagWhileReading(t *testing.T) {
	const writers = 50

	// Seed the first tag at construction time so the Tags container itself is
	// initialized before the goroutines start (lazy initialization of the
	// `Tags` FIELD is the caller's to sequence; the Set is what's safe for
	// concurrent use).
	err := New("shared error", WithTag("tag-00"))

	cE, ok := To(err)
	require.True(t, ok)

	var wg sync.WaitGroup

	// Writers: each applies its own tag via WithTag on the SAME instance
	// (re-adding "tag-00" concurrently is fine - it's a set).
	for i := range writers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			WithTag(fmt.Sprintf("tag-%02d", i))(cE)
		}()
	}

	// Readers: Error() and json.Marshal run concurrently with the writers.
	for range 10 {
		wg.Add(2)

		go func() {
			defer wg.Done()

			assert.NotEmpty(t, cE.Error())
		}()

		go func() {
			defer wg.Done()

			b, mErr := json.Marshal(cE)
			assert.NoError(t, mErr)
			assert.NotEmpty(t, b)
		}()
	}

	wg.Wait()

	// All tags must be present, and String() must render them sorted.
	require.NotNil(t, cE.Tags)
	assert.Equal(t, writers, cE.Tags.Size())

	expected := make([]string, 0, writers)

	for i := range writers {
		expected = append(expected, fmt.Sprintf("tag-%02d", i))
	}

	assert.Equal(t, strings.Join(expected, ", "), cE.Tags.String())
	assert.Contains(t, cE.Error(), "Tags: "+strings.Join(expected, ", "))
}

//////
// Happy path: a mixed concurrent workload over the locked Set API
// (Add/Remove/Contains/Values/Each/Size/Empty/String/Clear).
//////

func TestSetConcurrency_MixedAPI(t *testing.T) {
	const items = 50

	s := newSet()

	var wg sync.WaitGroup

	for i := range items {
		wg.Add(2)

		// Writer.
		go func() {
			defer wg.Done()

			s.Add(fmt.Sprintf("item-%02d", i))
		}()

		// Reader: none of these must race with the writers above.
		go func() {
			defer wg.Done()

			_ = s.Contains(fmt.Sprintf("item-%02d", i))
			_ = s.Values()
			_ = s.Size()
			_ = s.Empty()
			_ = s.String()

			s.Each(func(_ int, value interface{}) {
				assert.NotNil(t, value)
			})
		}()
	}

	wg.Wait()

	// Every item must have made it in exactly once.
	assert.Equal(t, items, s.Size())
	assert.False(t, s.Empty())
	assert.True(t, s.Contains("item-00", fmt.Sprintf("item-%02d", items-1)))
	assert.Len(t, s.Values(), items)

	// Concurrent removals of the first half.
	half := items / 2

	for i := range half {
		wg.Add(1)

		go func() {
			defer wg.Done()

			s.Remove(fmt.Sprintf("item-%02d", i))
		}()
	}

	wg.Wait()

	assert.Equal(t, items-half, s.Size())
	assert.False(t, s.Contains("item-00"))

	// Clear empties the set.
	s.Clear()

	assert.True(t, s.Empty())
	assert.Equal(t, 0, s.Size())
}

//////
// Edge case: Copy where src and target share the SAME Tags pointer must not
// deadlock (it would mean iterating a Set while adding to the very same Set).
//////

func TestSetConcurrency_CopySharedTagsNoDeadlock(t *testing.T) {
	shared := newSet("a", "b")

	src := &CustomError{Message: "source error", Tags: shared}
	target := &CustomError{Message: "target error", Tags: shared}

	done := make(chan struct{})

	go func() {
		defer close(done)

		_ = Copy(src, target)
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("Copy deadlocked when src and target share the same Tags pointer")
	}

	// The shared set is untouched and the copy semantics are preserved.
	assert.Equal(t, "a, b", shared.String())
	assert.Same(t, shared, target.Tags)
	assert.Equal(t, "source error", target.Message)
}

//////
// Edge case: String() determinism is preserved - tags added concurrently and
// out of order still render sorted ("a, b, c").
//////

func TestSetConcurrency_StringDeterminism(t *testing.T) {
	s := newSet()

	var wg sync.WaitGroup

	for _, tag := range []string{"c", "a", "b"} {
		wg.Add(1)

		go func() {
			defer wg.Done()

			s.Add(tag)
		}()
	}

	wg.Wait()

	assert.Equal(t, "a, b, c", s.String())
}

//////
// Bad path: a fresh (empty) Set must not panic - every locked method behaves
// sanely with no elements.
//////

func TestSetConcurrency_EmptySetDoesNotPanic(t *testing.T) {
	s := newSet()

	assert.NotPanics(t, func() {
		assert.True(t, s.Empty())
		assert.Equal(t, 0, s.Size())
		assert.Empty(t, s.Values())
		assert.Empty(t, s.String())
		assert.False(t, s.Contains("nope"))

		called := false

		s.Each(func(_ int, _ interface{}) {
			called = true
		})

		assert.False(t, called)

		// Mutators on an empty set are no-ops, not panics.
		s.Remove("nope")
		s.Clear()

		// JSON of an empty set is an empty array.
		b, err := json.Marshal(s)
		assert.NoError(t, err)
		assert.JSONEq(t, "[]", string(b))
	})
}
