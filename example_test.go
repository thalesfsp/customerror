// Copyright 2021 The customerror Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package customerror

import (
	"errors"
	"fmt"
	"net/http"
)

// Demonstrates how to create static, and dynamic custom errors, also how to
// check, and instrospect custom errors.
func ExampleNew() {
	// Custom static error definition.
	ErrMissingID := NewMissingError("id", WithCode("E1010"))

	// Some function, for demo purpose.
	SomeFunc := func(id string) error {
		if id == "" {
			// Usage of the custom static error.
			return ErrMissingID
		}

		// Dynamic custom error.
		return NewFailedToError("write to disk", WithCode("E1523"))
	}

	// Case: Without `id`, returns `ErrMissingID`.
	if err := SomeFunc(""); err != nil {
		fmt.Println(errors.Is(err, ErrMissingID)) // true

		var cE *CustomError
		if errors.As(err, &cE) {
			fmt.Println(cE.StatusCode) // 400
		}

		fmt.Println(err) // E1010: missing id (400 - Bad Request)
	}

	// Case: With `id`, returns dynamic error.
	if err := SomeFunc("12345"); err != nil {
		var cE *CustomError
		if errors.As(err, &cE) {
			fmt.Println(cE.StatusCode) // 500
		}

		fmt.Println(err) // E1523: failed to write to disk (500 - Internal Server Error)
	}

	// output:
	// true
	// 400
	// E1010: missing id
	// 500
	// E1523: failed to write to disk
}

// Demonstrates how to create static, and dynamic custom errors, also how to
// check, and instrospect custom errors.
//
//nolint:errorlint,forcetypeassert
func ExampleNew_options() {
	fmt.Println(
		NewMissingError("id", WithCode("E1010"), WithStatusCode(http.StatusNotAcceptable), WithError(errors.New("some error"))).(*CustomError).APIError(),
	)

	// output:
	// E1010: missing id (406 - Not Acceptable). Original Error: some error
}

// Demonstrates error chain. `errB` will wrap `errA` and will be considered the
// same by propagating the chain.
func ExampleNew_is() {
	errA := NewMissingError("id")
	errB := NewMissingError("name", WithError(errA))

	fmt.Println(errors.Is(errB, errA))

	// output:
	// true
}
