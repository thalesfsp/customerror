// Copyright 2021 The customerror Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package customerror

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func checkIfStringContainsMany(s string, subs ...string) []string {
	missing := []string{}

	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			missing = append(missing, sub)
		}
	}

	return missing
}

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
	fmt.Println(NewMissingError("id", WithError(errors.New("hehehe")), WithIgnoreString("hehehe")) == nil)
	fmt.Println(NewMissingError("id", WithIgnoreString("hahaha")) == nil)

	// output:
	// true
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

// Demonstrates the WithTag option.
//
//nolint:errorlint,forcetypeassert
func ExampleNew_optionsWithTag() {
	fmt.Println(NewMissingError(
		"id",
		WithTag("test1", "test2"),
		WithCode("E1010"),
		WithStatusCode(http.StatusNotAcceptable),
		WithError(errors.New("some error")),
	))

	fmt.Println(NewMissingError(
		"id",
		WithTag("test1", "test2"),
		WithCode("E1010"),
		WithStatusCode(http.StatusNotAcceptable),
		WithError(errors.New("some error")),
	).(*CustomError).APIError())

	// output:
	// E1010: missing id. Original Error: some error. Tags: test1, test2
	// E1010: missing id (406 - Not Acceptable). Original Error: some error. Tags: test1, test2
}

// Demonstrates the WithFields option.
//
//nolint:errorlint,forcetypeassert
func ExampleNew_optionsWithFields() {
	fmt.Println(NewMissingError(
		"id",
		WithTag("test1", "test2"),
		WithField(map[string]interface{}{
			"testKey1": "testValue1",
			"testKey2": "testValue2",
		}),
		WithCode("E1010"),
		WithStatusCode(http.StatusNotAcceptable),
		WithError(errors.New("some error")),
	))

	fmt.Println(NewMissingError(
		"id",
		WithTag("test1", "test2"),
		WithField(map[string]interface{}{
			"testKey1": "testValue1",
			"testKey2": "testValue2",
		}),
		WithCode("E1010"),
		WithStatusCode(http.StatusNotAcceptable),
		WithError(errors.New("some error")),
	).(*CustomError).APIError())

	// output:
	// E1010: missing id. Original Error: some error. Tags: test1, test2. Fields: testKey1=testValue1, testKey2=testValue2
	// E1010: missing id (406 - Not Acceptable). Original Error: some error. Tags: test1, test2. Fields: testKey1=testValue1, testKey2=testValue2
}

func ExampleNew_NewFactory() {
	factory := NewFactory(
		map[string]interface{}{
			"test1": "test2",
			"test3": "test4",
		},
		"testTag1", "testTag2", "testTag3",
	)

	childFactory := factory.NewChildError(
		map[string]interface{}{
			"test1": "test2",
			"test5": "test6",
		},
		"testTag2", "testTag3", "testTag4",
	)

	// Write to a buffer and check the output.
	var buf bytes.Buffer

	fmt.Fprint(&buf, childFactory.NewMissingError("id"))
	fmt.Fprint(&buf, childFactory.NewFailedToError("insert id"))
	fmt.Fprint(&buf, childFactory.NewInvalidError("id"))
	fmt.Fprint(&buf, childFactory.NewMissingError("id"))
	fmt.Fprint(&buf, childFactory.NewRequiredError("id"))
	fmt.Fprint(&buf, childFactory.NewHTTPError(400))

	finalMessage := buf.String()

	fmt.Println(len(checkIfStringContainsMany(
		finalMessage,
		"missing id", "failed to insert id", "invalid id", "missing id", "id required", "bad request",
		"Tags:", "Fields:", "testTag1", "testTag2", "testTag3", "testTag4",
		"test1=test2", "test3=test4", "test5=test6",
	)) == 0)

	// output:
	// true
}
