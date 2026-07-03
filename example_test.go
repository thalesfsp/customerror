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
	"time"

	"github.com/eapache/go-resiliency/retrier"
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
	ErrMissingID := NewMissingError("id", WithErrorCode("E1010"))

	// Some function, for demo purpose.
	SomeFunc := func(id string) error {
		if id == "" {
			// Usage of the custom static error.
			return ErrMissingID
		}

		// Dynamic custom error.
		return NewFailedToError("write to disk", WithErrorCode("E1523"))
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
		NewMissingError("id", WithErrorCode("E1010"), WithStatusCode(http.StatusNotAcceptable), WithError(errors.New("some error"))).(*CustomError).APIError(),
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
	fmt.Println(New("", WithStatusCode(http.StatusAccepted), WithErrorCode("E1010")))
	fmt.Println(New("", WithStatusCode(http.StatusAccepted)).(*CustomError).APIError())
	fmt.Println(New("", WithStatusCode(http.StatusAccepted), WithErrorCode("E1010")).(*CustomError).APIError())

	fmt.Println(New("", WithErrorCode("E1010")))
	fmt.Println(New("", WithErrorCode("E1010"), WithStatusCode(http.StatusAccepted)))
	fmt.Println(New("", WithErrorCode("E1010")).(*CustomError).APIError())
	fmt.Println(New("", WithErrorCode("E1010"), WithStatusCode(http.StatusAccepted)).(*CustomError).APIError())

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
		WithErrorCode("E1010"),
		WithStatusCode(http.StatusNotAcceptable),
		WithError(errors.New("some error")),
	))

	fmt.Println(NewMissingError(
		"id",
		WithTag("test1", "test2"),
		WithErrorCode("E1010"),
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
	// Store message in a buf to be checked later.
	var buf strings.Builder

	fmt.Fprintln(&buf, NewMissingError(
		"id",
		WithTag("test1", "test2"),
		WithFields(map[string]interface{}{
			"testKey1": "testValue1",
			"testKey2": "testValue2",
		}),
		WithErrorCode("E1010"),
		WithStatusCode(http.StatusNotAcceptable),
		WithError(errors.New("some error")),
	))

	fmt.Fprintln(&buf, NewMissingError(
		"id",
		WithTag("test1", "test2"),
		WithFields(map[string]interface{}{
			"testKey1": "testValue1",
			"testKey2": "testValue2",
		}),
		WithErrorCode("E1010"),
		WithStatusCode(http.StatusNotAcceptable),
		WithError(errors.New("some error")),
	).(*CustomError).APIError())

	fmt.Println(len(checkIfStringContainsMany(
		buf.String(),
		"E1010: missing id",
		"E1010: missing id (406 - Not Acceptable)",
		"Original Error: some error",
		"Tags: test1, test2",
		"Fields:",
		"testKey1=testValue1",
		"testKey2=testValue2",
	)) == 0)

	// output:
	// true
}

// Demonstrates the WithField option.
//
//nolint:errorlint,forcetypeassert
func ExampleNew_optionsWithField() {
	// Store message in a buf to be checked later.
	var buf strings.Builder

	fmt.Fprintln(&buf, NewMissingError(
		"id",
		WithTag("test1", "test2"),
		WithField("testKey1", "testValue1"),
		WithField("testKey2", "testValue2"),
		WithErrorCode("E1010"),
		WithStatusCode(http.StatusNotAcceptable),
		WithError(errors.New("some error")),
	))

	fmt.Fprintln(&buf, NewMissingError(
		"id",
		WithTag("test1", "test2"),
		WithField("testKey1", "testValue1"),
		WithField("testKey2", "testValue2"),
		WithErrorCode("E1010"),
		WithStatusCode(http.StatusNotAcceptable),
		WithError(errors.New("some error")),
	).(*CustomError).APIError())

	fmt.Println(len(checkIfStringContainsMany(
		buf.String(),
		"E1010: missing id",
		"E1010: missing id (406 - Not Acceptable)",
		"Original Error: some error",
		"Tags: test1, test2",
		"Fields:",
		"testKey1=testValue1",
		"testKey2=testValue2",
	)) == 0)

	// output:
	// true
}

// ExampleNew_NewFactory demonstrates the usage of the NewFactory function.
func ExampleNew_newFactory() {
	factory := Factory(
		"id",
		WithFields(map[string]interface{}{
			"test1": "test2",
			"test3": "test4",
		}),
		WithTag("testTag1", "testTag2", "testTag3"),
	)

	childFactory := factory.NewChildError(
		WithFields(map[string]interface{}{
			"test1": "test2",
			"test5": "test6",
		}),
		WithTag("testTag2", "testTag3", "testTag4"),
	)

	// Write to a buffer and check the output.
	var buf bytes.Buffer

	fmt.Fprint(&buf, childFactory.NewMissingError())
	fmt.Fprint(&buf, childFactory.NewFailedToError())
	fmt.Fprint(&buf, childFactory.NewInvalidError())
	fmt.Fprint(&buf, childFactory.NewRequiredError())
	fmt.Fprint(&buf, childFactory.NewHTTPError(400))

	finalMessage := buf.String()

	missing := checkIfStringContainsMany(
		finalMessage,
		"missing id", "failed to id", "invalid id", "missing id", "id required", "bad request",
		"Tags:", "Fields:", "testTag1", "testTag2", "testTag3", "testTag4",
		"test1=test2", "test3=test4", "test5=test6",
	)

	if len(missing) == 0 {
		fmt.Println(true)
	} else {
		fmt.Println(missing)
	}

	// output:
	// true
}

// ExampleNew_i18n demonstrates how to create an error catalog with translations
// and how to throw errors in different languages.
//
// SEE: `i18n.md` file for more information.
func ExampleNew_i18n() {
	//////
	// The following, is usually defined in the `errorcatalog.go` file.
	//////

	const (
		//////
		// Define the error code constant. It helps to identify the error in
		// systems like Elasticsearch, Splunk and Datadog. It also helps to
		// maintain consistency across the application.
		//////

		ErrInvalidHardDrivePath = "ERR_INVALID_HARD_DRIVE_PATH"
	)

	// Create the application error catalog.
	catalog := MustNewCatalog("myIncredibleApp").
		// Add errors by their constant error codes while setting up the
		// translations. For Spanish and French, it uses the built-in list of
		// common languages instead of hardcoding the language.
		//
		// NOTE: For supported built-in languages, the default word(s) used by
		// the built-in error functions (example: `NewFailedToError`), are
		// automatically translated and included in the message.
		//
		// SEE: `languages.go` file an up-to-date list of the supported built-in
		// list of languages.
		//
		// For ANY other language there are two options:
		// 1. Don't use the built-in functions but instead use the `New` function
		// and write the message in full, for example: "invalid hard drive path".
		// 2. Setup the language (see `ExampleNew_i18nSetupNewLang`).
		//
		// Reason: It's impossible for any package to cover all the possible
		// languages, combinations, and their translations.
		//
		// No need to add the "invalid" word.
		MustSet(ErrInvalidHardDrivePath, "hard drive path",
			// No need to add the "invalid" word.
			WithTranslation(Spanish.String(), "ruta de disco duro"),

			// No need to add the "invalid" word.
			WithTranslation(French.String(), "chemin de disque dur"),
		)

	//////
	// The following, from anywhere in the application.
	//////

	// Retrieve the error from the catalog.
	err := catalog.MustGet(ErrInvalidHardDrivePath)

	// Throw that in the Spanish language as an and using the built-in
	// `InvalidError` function which automatically sets the HTTP status code to
	// `StatusBadRequest`. The language is specified by using the built-in list
	// of languages. The default "invalid" word is automatically translated.
	//
	// SEE: `languages.go` file for the list of languages.
	fmt.Println(err.NewInvalidError(WithLanguage(Spanish.String())))

	// The same, but in French.
	fmt.Println(err.NewInvalidError(WithLanguage(French.String())))

	// The same, standard way - in English.
	fmt.Println(err.NewInvalidError())

	// output:
	// ERR_INVALID_HARD_DRIVE_PATH: ruta de disco duro inválido
	// ERR_INVALID_HARD_DRIVE_PATH: chemin de disque dur invalide
	// ERR_INVALID_HARD_DRIVE_PATH: invalid hard drive path
}

// ExampleNew_i18n demonstrates how to create an error catalog with translations
// how to throw errors in different languages, and how to setup a new language.
//
// SEE: `i18n.md` file for more information.
func ExampleNew_i18nSetupNewLang() {
	//////
	// The following, is usually defined in the `errorcatalog.go` file.
	//////

	// Define a constant for the new language, to help with consistency.
	const Japanase = "jp"

	// Let's pretend that beyond English, Spanish, and French, the
	// application must support Japanese. In this case, Japanase is not part
	// of the built-in list of languages. We start by setting up the language,
	// and the respective built-in functions error messages.
	//
	// NOTE: MustAddNewLanguage properly updates the internal, package-level,
	// singleton.
	//
	// WARN: If you use `MustAddNewLanguage`, and specify an invalid language
	// such as "asd", it will panic! The language must be a valid ISO 639-1 or
	// ISO 3166-1 alpha-2.
	MustAddNewLanguage("jp", NewErrorPrefixMap(
		// "failed to" template.
		"%s に失敗しました",

		// "invalid" template.
		"%s が無効です",

		// "missing" template.
		"%s が見つかりません",

		// "required" template.
		"%s が必要です",

		// "not found" template.
		"%s が見つかりませんでした",
	))

	const (
		//////
		// Define the error code constant. It helps to identify the error in
		// systems like Elasticsearch, Splunk and Datadog. It also helps to
		// maintain consistency across the application.
		//////

		ErrInvalidHardDrivePath = "ERR_INVALID_HARD_DRIVE_PATH"
	)

	// Create the application error catalog.
	catalog := MustNewCatalog("myIncredibleApp").
		// Add errors by their constant error codes while setting up the
		// translations. For Spanish and French, it uses the built-in list of
		// common languages instead of hardcoding the language.
		//
		// NOTE: For supported built-in languages, the default word(s) used by
		// the built-in error functions (example: `NewFailedToError`), are
		// automatically translated and included in the message.
		//
		// SEE: `languages.go` file an up-to-date list of the supported built-in
		// list of languages.
		//
		// For ANY other language there are two options:
		// 1. Don't use the built-in functions but instead use the `New` function
		// and write the message in full, for example: "invalid hard drive path".
		// 2. Setup the language (see `ExampleNew_i18nSetupNewLang`).
		//
		// Reason: It's impossible for any package to cover all the possible
		// languages, combinations, and their translations.
		//
		// No need to add the "invalid" word.
		MustSet(ErrInvalidHardDrivePath, "hard drive path",
			// No need to add the "invalid" word.
			WithTranslation(Spanish.String(), "ruta de disco duro"),

			// No need to add the "invalid" word.
			WithTranslation(French.String(), "chemin de disque dur"),

			// Added support, no need to add the "invalid" word.
			WithTranslation("jp", "ハードドライブのパス"),
		)

	// Retrieve the error from the catalog.
	err := catalog.MustGet(ErrInvalidHardDrivePath)

	// Throw that in the Spanish language as an and using the built-in
	// `InvalidError` function which automatically sets the HTTP status code to
	// `StatusBadRequest`. The language is specified by using the built-in list
	// of languages. The default "invalid" word is automatically translated.
	//
	// SEE: `languages.go` file for the list of languages.
	fmt.Println(err.NewInvalidError(WithLanguage(Spanish.String())))

	// The same, but in French.
	fmt.Println(err.NewInvalidError(WithLanguage(French.String())))

	// The same, but in Japanese.
	fmt.Println(err.NewInvalidError(WithLanguage(Japanase)))

	// The same, standard way - in English.
	fmt.Println(err.NewInvalidError())

	// output:
	// ERR_INVALID_HARD_DRIVE_PATH: ruta de disco duro inválido
	// ERR_INVALID_HARD_DRIVE_PATH: chemin de disque dur invalide
	// ERR_INVALID_HARD_DRIVE_PATH: ハードドライブのパス が無効です
	// ERR_INVALID_HARD_DRIVE_PATH: invalid hard drive path
}

type CustomClassifier struct{}

// Classify implements the Classifier interface.
func (hSCC CustomClassifier) Classify(err error) retrier.Action {
	// Should do nothing if there's no error.
	if err == nil {
		return retrier.Succeed
	}

	//////
	// Cast error into customerror to evaluate if the error is retryable.
	//////

	var cE *CustomError

	if errors.As(err, &cE) {
		if cE.Retryable {
			// Update the error to be retried.
			cE.SetRetried(true)

			return retrier.Retry
		}
	}

	// Should fail for everything else.
	return retrier.Fail
}

// ExampleNew_Retryable error and SetRetried function. In this example the go
// go-resiliency/retrier package will be used, but any other retry package
// should work as well. Additionally, the example demonstrates how to check if
// the error is retryable, if it has a specific error code, if it has a specific
// HTTP status code, and if it is a custom error.
func ExampleNew_newRetryableError() {
	// Create a custom retryable error.
	retryableCE := NewFailedToError(
		"write to disk",
		WithErrorCode("E1523"),
		WithRetryable(true),
	)

	// Initialize a retrier with exponential backoff strategy.
	r1 := retrier.New(
		retrier.ConstantBackoff(1, 300*time.Millisecond),
		CustomClassifier{},
	)

	// Execute the request with retry logic.
	if err := r1.Run(func() error {
		// Throw the retryable error.
		return retryableCE
	}); err != nil {
		fmt.Println(err)
	}

	fmt.Println(IsRetryable(retryableCE))
	fmt.Println(IsErrorCode(retryableCE, "E1523"))
	fmt.Println(IsHTTPStatus(retryableCE, http.StatusInternalServerError))
	fmt.Println(IsCustomError(retryableCE))

	if toCE, ok := To(retryableCE); ok {
		fmt.Println(toCE.StatusCode)
	}

	// output:
	// E1523: failed to write to disk. Retryable: true. Retried: true
	// true
	// true
	// true
	// true
	// 500
}
