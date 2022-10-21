// Copyright 2021 The customerror Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package customerror

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
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

// Demonstrates JSON marshalling of custom errors.
func ExampleNew_marshalJSON() {
	// New buffer string.
	var buf strings.Builder

	errA := NewMissingError("id")
	errB := NewMissingError("name", WithError(errA))

	if err := json.NewEncoder(&buf).Encode(errB); err != nil {
		panic(err)
	}

	fmt.Println(strings.Contains(buf.String(), `message":"missing name. Original Error: missing id`))

	// output:
	// true
}

// Demonstrates the WithIgnoreString option.
func ExampleNew_optionsWithIgnoreString() {
	fmt.Println(NewMissingError("id", WithIgnoreString("id")) == nil)
	fmt.Println(NewMissingError("id", WithIgnoreString("hahaha")) == nil)

	// output:
	// true
	// false
}

// Demonstrates the WithIgnoreFunc option.
func ExampleNew_optionsWithIgnoreIf() {
	fmt.Println(NewMissingError("id", WithIgnoreFunc(func(cE *CustomError) bool {
		return strings.Contains(cE.Message, "id")
	})) == nil)

	// output:
	// true
}

// Demonstrates the NewHTTPError custom error.
//
//nolint:errorlint,forcetypeassert
func ExampleNew_newHTTPError() {
	fmt.Println(NewHTTPError(http.StatusNotFound).(*CustomError).APIError())
	fmt.Println(NewHTTPError(http.StatusNotFound).(*CustomError).Error())
	fmt.Println(NewHTTPError(http.StatusNotFound))

	// output:
	// not found (404 - Not Found)
	// not found
	// not found
}

// Demonstrates errors without message but with status code.
//
//nolint:errorlint,forcetypeassert
func ExampleNew_newNoMessage() {
	fmt.Println(New("", WithStatusCode(http.StatusAccepted)))
	fmt.Println(New("", WithStatusCode(http.StatusAccepted), WithCode("E1010")))
	fmt.Println(New("", WithStatusCode(http.StatusAccepted)).(*CustomError).APIError())
	fmt.Println(New("", WithStatusCode(http.StatusAccepted), WithCode("E1010")).(*CustomError).APIError())

	fmt.Println(New("", WithCode("E1010")))
	fmt.Println(New("", WithCode("E1010"), WithStatusCode(http.StatusAccepted)))
	fmt.Println(New("", WithCode("E1010")).(*CustomError).APIError())
	fmt.Println(New("", WithCode("E1010"), WithStatusCode(http.StatusAccepted)).(*CustomError).APIError())

	// output:
	// Accepted
	// E1010: Accepted
	// Accepted (202)
	// E1010: Accepted (202)
	// E1010
	// E1010: Accepted
	// E1010
	// E1010: Accepted (202)
}
