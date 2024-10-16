// Copyright 2021 The customerror Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package customerror

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/emirpasic/gods/sets/treeset"
	"github.com/go-playground/validator/v10"
)

//////
// Helpers.
//////

// Copy performs a deep copy of the src CustomError into the target CustomError.
// It ensures that all fields, including nested structures like maps and sets,
// are properly copied. This function is useful when you need to create a new
// CustomError instance based on an existing one, while avoiding any shared
// references to mutable fields.
func Copy(src, target *CustomError) *CustomError {
	if src.Code != "" {
		target.Code = src.Code
	}

	if src.Retryable {
		target.Retryable = src.Retryable
	}

	if src.Retried {
		target.Retried = src.Retried
	}

	if src.Err != nil {
		target.Err = src.Err
	}

	if src.language != "" {
		target.language = src.language
	}

	if src.Message != "" {
		target.Message = src.Message
	}

	if src.StatusCode != 0 {
		target.StatusCode = src.StatusCode
	}

	if src.ignore {
		target.ignore = src.ignore
	}

	// Merge the language messages.
	if src.LanguageMessageMap != nil {
		if target.LanguageMessageMap == nil {
			target.LanguageMessageMap = &sync.Map{}
		}

		finalLanguageMessageMap := &sync.Map{}

		src.LanguageMessageMap.Range(func(key, value interface{}) bool {
			finalLanguageMessageMap.Store(key, value)

			return true
		})

		target.LanguageMessageMap.Range(func(key, value interface{}) bool {
			finalLanguageMessageMap.Store(key, value)

			return true
		})

		target.LanguageMessageMap = finalLanguageMessageMap
	}

	// Merge fields.
	if src.Fields != nil {
		if target.Fields == nil {
			target.Fields = &sync.Map{}
		}

		finalFields := &sync.Map{}

		src.Fields.Range(func(key, value interface{}) bool {
			finalFields.Store(key, value)

			return true
		})

		target.Fields.Range(func(key, value interface{}) bool {
			finalFields.Store(key, value)

			return true
		})

		target.Fields = finalFields
	}

	// Merge the tags.
	if src.Tags != nil {
		if target.Tags == nil {
			target.Tags = &Set{treeset.NewWithStringComparator()}
		}

		src.Tags.Each(func(_ int, value interface{}) {
			target.Tags.Add(value)
		})
	}

	return target
}

// Process fields and add them to the error message.
func processFields(
	errMsg string,
	fields *sync.Map,
) string {
	if fields != nil {
		errMsg = fmt.Sprintf("%s. Fields:", errMsg)

		fields.Range(func(k, v interface{}) bool {
			errMsg = fmt.Sprintf("%s %s=%v,", errMsg, k, v)

			return true
		})

		errMsg = strings.TrimSuffix(errMsg, ",")
	}

	return errMsg
}

// mapToSyncMap converts a map to a sync.Map.
func mapToSyncMap(m map[string]interface{}) *sync.Map {
	sm := &sync.Map{}

	for k, v := range m {
		sm.Store(k, v)
	}

	return sm
}

// syncMapToMap converts a sync.Map to a map.
func syncMapToMap(sm *sync.Map) map[string]interface{} {
	m := make(map[string]interface{})

	if sm != nil {
		sm.Range(func(k, v interface{}) bool {
			if str, ok := k.(string); ok {
				m[str] = v
			}

			return true
		})
	}

	return m
}

// Set is a wrapper around the treeset.Set, providing a collection
// that stores unique elements in a sorted order. It is used in the
// CustomError struct to maintain a sorted set of tags.
type Set struct {
	*treeset.Set
}

// String implements the Stringer interface for the Set type.
// It returns a comma-separated string representation of all elements
// in the set, useful for debugging and error message formatting.
func (s *Set) String() string {
	items := []string{}

	s.Each(func(_ int, value interface{}) {
		items = append(items, fmt.Sprintf("%v", value))
	})

	return strings.Join(items, ", ")
}

// CustomError is the base block to create custom errors. It provides context -
// a `Message` to an optional `Err`. Additionally a `Code` - for example "E1010",
// and `StatusCode` can be provided.
type CustomError struct {
	// Code can be any custom code, e.g.: E1010.
	Code string `json:"code,omitempty" validate:"omitempty,gte=2"`

	// Err optionally wraps the original error.
	Err error `json:"-"`

	// Field enhances the error message with more structured information.
	Fields *sync.Map `json:"fields,omitempty"`

	// Human readable message. Minimum length: 3.
	Message string `json:"message" validate:"required,gte=3"`

	// Message in different languages.
	LanguageMessageMap LanguageMessageMap `json:"languageMessageMap"`

	// LanguageErrorTypeMap is a map of language prefixes to templates such
	// as "missing %s", "%s required", "%s invalid", etc.
	LanguageErrorTypeMap LanguageErrorMap `json:"languageErrorTypeMap"`

	// Retryable indicates if the error is retryable.
	Retryable bool `json:"retryable"`

	// Retried indicates if the error has been retried.
	Retried bool `json:"retried"`

	// StatusCode is a valid HTTP status code, e.g.: 404.
	StatusCode int `json:"-" validate:"omitempty,gte=100,lte=511"`

	// Tags is a SET of tags which helps to categorize the error.
	Tags *Set `json:"tags,omitempty"`

	// If set to true, the error will be ignored (return nil).
	ignore bool `json:"-"`

	// Language to be use for the message and prefix.
	language Language
}

//////
// Error interface implementation.
//////

// Error interface implementation returns the properly formatted error message.
// It will contain `Code`, `Tags`, `Fields` and any wrapped error.
func (cE *CustomError) Error() string {
	errMsg := cE.Message

	if cE.Code != "" {
		if cE.Message != cE.Code {
			errMsg = fmt.Sprintf("%s: %s", cE.Code, errMsg)
		} else {
			errMsg = cE.Code
		}
	}

	if cE.Err != nil {
		errMsg = fmt.Errorf("%s. Original Error: %w", errMsg, cE.Err).Error()
	}

	if cE.Tags != nil {
		errMsg = fmt.Sprintf("%s. Tags: %s", errMsg, cE.Tags.String())
	}

	errMsg = processFields(errMsg, cE.Fields)

	if cE.Retryable {
		errMsg = fmt.Sprintf("%s. Retryable: %t. Retried: %t", errMsg, cE.Retryable, cE.Retried)
	}

	return errMsg
}

// Is interface implementation ensures chain continuity. Treats `CustomError` as
// equivalent to `err`.
//
// SEE https://blog.golang.org/go1.13-errors
//
//nolint:errorlint
func (cE *CustomError) Is(err error) bool {
	return cE.Err == err
}

// Unwrap interface implementation returns inner error.
func (cE *CustomError) Unwrap() error {
	return cE.Err
}

//////
// Implementing the json.Marshaler interface.
//////

// MarshalJSON implements the json.Marshaler interface.
//
// SEE https://gist.github.com/thalesfsp/3a1252530750e2370345a2418721ff54
func (cE *CustomError) MarshalJSON() ([]byte, error) {
	// Define a temporary map that matches the desired JSON format.
	temp := make(map[string]interface{})

	// Populate the temporary map.
	temp["message"] = cE.JustError()

	if cE.Code != "" {
		temp["code"] = cE.Code
	}

	if cE.Tags != nil && !cE.Tags.Empty() {
		temp["tags"] = cE.Tags
	}

	if cE.Retryable {
		temp["retryable"] = cE.Retryable
	}

	if cE.Retried {
		temp["retried"] = cE.Retried
	}

	if cE.Fields != nil {
		// Convert the sync.Map to a regular map so that we can iterate over its keys.
		fields := syncMapToMap(cE.Fields)

		// Populate the fields of the temporary map.
		if len(fields) > 0 {
			for k, v := range fields {
				if k != "" && v != nil {
					temp[k] = v
				}
			}
		}
	}

	if cE.StatusCode > 0 {
		temp["statusCode"] = cE.StatusCode
	}

	// Serialize the temporary map to JSON.
	return json.Marshal(temp)
}

//////
// Error message formatting.
//////

// JustError returns the error message without any additional information.
func (cE *CustomError) JustError() string {
	errMsg := cE.Message

	if cE.Err != nil {
		errMsg = fmt.Errorf("%s. Original Error: %w", errMsg, cE.Err).Error()
	}

	return errMsg
}

// APIError is like error plus status code information.
func (cE *CustomError) APIError() string {
	errMsg := cE.Message

	if cE.Code != "" {
		if cE.Message != cE.Code {
			errMsg = fmt.Sprintf("%s: %s", cE.Code, errMsg)
		} else {
			errMsg = cE.Code
		}
	}

	if cE.StatusCode != 0 {
		if cE.Message != http.StatusText(cE.StatusCode) {
			errMsg = fmt.Sprintf("%s (%d - %s)", errMsg, cE.StatusCode, http.StatusText(cE.StatusCode))
		} else {
			errMsg = fmt.Sprintf("%s (%d)", errMsg, cE.StatusCode)
		}
	}

	if cE.Err != nil {
		errMsg = fmt.Errorf("%s. Original Error: %w", errMsg, cE.Err).Error()
	}

	if cE.Tags != nil {
		errMsg = fmt.Sprintf("%s. Tags: %s", errMsg, cE.Tags.String())
	}

	errMsg = processFields(errMsg, cE.Fields)

	if cE.Retryable {
		errMsg = fmt.Sprintf("%s. Retryable: %t. Retried: %t", errMsg, cE.Retryable, cE.Retried)
	}

	return errMsg
}

// FormatError formats the error message with the given error type.
func (cE *CustomError) FormatError(errorType string, opts ...Option) *CustomError {
	if cE == nil {
		return nil
	}

	finalCE := &CustomError{}

	finalCE = Copy(cE, finalCE)

	// Apply options.
	for _, opt := range opts {
		opt(finalCE)
	}

	if finalCE.language != "" {
		template, err := GetTemplate(string(finalCE.language), string(FailedTo))
		if err != nil {
			template2, err := GetTemplate(finalCE.language.GetRoot(), errorType)
			if err != nil {
				panic(err)
			}

			template = template2
		}

		finalCE.Message = fmt.Sprintf(template, finalCE.Message)

		return finalCE
	}

	return finalCE
}

//////
// Factory methods.
//////

// NewFailedToError is the building block for errors usually thrown when some
// action failed, e.g: "Failed to create host". Default status code is `500`.
//
// NOTE: Preferably don't use with the `WithLanguage` because of the "Failed to"
// part. Prefer to use `New` instead.
//
// NOTE: Status code can be redefined, call `SetStatusCode`.
func (cE *CustomError) NewFailedToError(opts ...Option) error {
	finalCE := cE.FormatError(string(FailedTo), opts...)

	if finalCE.language == "" {
		finalCE = Copy(NewFailedToError(finalCE.Message, opts...).(*CustomError), finalCE)

		return finalCE
	}

	return finalCE
}

// NewInvalidError is the building block for errors usually thrown when
// something fail validation, e.g: "Invalid port". Default status code is `400`.
//
// NOTE: Preferably don't use with the `WithLanguage` because of the "Invalid"
// part. Prefer to use `New` instead.
//
// NOTE: Status code can be redefined, call `SetStatusCode`.
func (cE *CustomError) NewInvalidError(opts ...Option) error {
	finalCE := cE.FormatError(string(Invalid), opts...)

	if finalCE.language == "" {
		finalCE = Copy(NewInvalidError(finalCE.Message, opts...).(*CustomError), finalCE)

		return finalCE
	}

	return finalCE
}

// NewMissingError is the building block for errors usually thrown when required
// information is missing, e.g: "Missing host". Default status code is `400`.
//
// NOTE: Preferably don't use with the `WithLanguage` because of the "Missing"
// part. Prefer to use `New` instead.
//
// NOTE: Status code can be redefined, call `SetStatusCode`.
func (cE *CustomError) NewMissingError(opts ...Option) error {
	finalCE := cE.FormatError(Missing.String(), opts...)

	if finalCE.language == "" {
		finalCE = Copy(NewMissingError(finalCE.Message, opts...).(*CustomError), finalCE)

		return finalCE
	}

	return finalCE
}

// NewRequiredError is the building block for errors usually thrown when
// required information is missing, e.g: "Port is required". Default status code is `400`.
//
// NOTE: Preferably don't use with the `WithLanguage` because of the "Required"
// part. Prefer to use `New` instead.
//
// NOTE: Status code can be redefined, call `SetStatusCode`.
func (cE *CustomError) NewRequiredError(opts ...Option) error {
	finalCE := cE.FormatError(Required.String(), opts...)

	if finalCE.language == "" {
		finalCE = Copy(NewRequiredError(finalCE.Message, opts...).(*CustomError), finalCE)

		return finalCE
	}

	return finalCE
}

// NewHTTPError is the building block for simple HTTP errors, e.g.: Not Found.
//
// NOTE: `WithLanguage` has no effect on it because of it's just a simple HTTP
// error.
//
// NOTE: Status code can be redefined, call `SetStatusCode`.
func (cE *CustomError) NewHTTPError(statusCode int, opts ...Option) error {
	if cE == nil {
		return nil
	}

	if cE.StatusCode == 0 {
		cE.StatusCode = statusCode
	}

	finalCE := &CustomError{}

	finalCE = Copy(cE, finalCE)

	httpCE := NewHTTPError(finalCE.StatusCode, opts...).(*CustomError)

	finalErrorMessage := httpCE.Message

	// Apply options.
	for _, opt := range opts {
		opt(finalCE)
	}

	finalCE.Message = finalErrorMessage

	finalCE = Copy(httpCE, finalCE)

	return finalCE
}

// New is the building block for other errors. Preferred method to be used for
// translations (WithLanguage).
func (cE *CustomError) New(opts ...Option) error {
	if cE == nil {
		return nil
	}

	finalCE := &CustomError{}

	finalCE = Copy(cE, finalCE)

	// Apply options.
	for _, opt := range opts {
		opt(finalCE)
	}

	finalCE = Copy(New(finalCE.Message, opts...).(*CustomError), finalCE)

	return finalCE
}

//////
// Exported functionalities.
//////

// Wrap `customError` around `errors`.
func Wrap(customError error, errors ...error) error {
	errMsgs := []string{}

	for _, err := range errors {
		if err != nil {
			errMsgs = append(errMsgs, err.Error())
		}
	}

	return fmt.Errorf("%w. Wrapped Error(s): %s", customError, strings.Join(errMsgs, ". "))
}

// NewChildError creates a new `CustomError` with the same fields and tags of
// the parent `CustomError` plus the new fields and tags passed as arguments.
func (cE *CustomError) NewChildError(opts ...Option) *CustomError {
	childCE := &CustomError{}

	// Apply the options.
	for _, opt := range opts {
		opt(childCE)
	}

	return Copy(cE, childCE)
}

// SetMessage sets the message of the error.
func (cE *CustomError) SetMessage(message string) {
	cE.Message = message
}

// SetRetried sets if the error has been retried.
func (cE *CustomError) SetRetried(status bool) {
	cE.Retried = status
}

//////
// Factory.
//////

// Base newInternal.
func newInternal(opts ...Option) *CustomError {
	cE := &CustomError{}

	// Apply options.
	for _, opt := range opts {
		opt(cE)
	}

	// Should use status code if no message is set. Status code should be
	// priority.
	if cE.Message == "" && cE.StatusCode > 0 {
		cE.Message = http.StatusText(cE.StatusCode)
	} else if cE.Message == "" && cE.Code != "" {
		cE.Message = cE.Code
	}

	// Should be able to programatically ignore errors (`WithIgnoreFunc`).
	if cE.ignore {
		return nil
	}

	return cE
}

// New creates a new validated custom error returning it as en `error`.
//
//nolint:revive
func New(message string, opts ...Option) error {
	cE := newInternal(prependOptions(opts, WithMessage(message))...)

	if cE == nil {
		return nil
	}

	if err := validator.New().Struct(cE); err != nil {
		if os.Getenv("CUSTOMERROR_ENVIRONMENT") == "testing" {
			log.Panicf("Invalid custom error. %s\n", err)
		} else {
			log.Fatalf("Invalid custom error. %s\n", err)
		}

		return nil
	}

	return cE
}

// Factory creates a validated and pre-defined error to be recalled and thrown
// later, with or without options. Possible options are:
// - `NewFailedToError`
// - `NewInvalidError`
// - `NewMissingError`
// - `NewRequiredError`
// - `NewHTTPError`.
func Factory(message string, opts ...Option) *CustomError {
	return newInternal(prependOptions(opts, WithMessage(message))...)
}

// IsCustomError checks if the error is a `CustomError`.
func IsCustomError(err error) bool {
	_, ok := err.(*CustomError)

	return ok
}

// To converts the error to a `CustomError`.
func To(err error) *CustomError {
	cE, ok := err.(*CustomError)

	if !ok {
		return nil
	}

	return cE
}

// IsHTTPStatus checks if the error is a `CustomError` with the
// specified HTTP status code.
func IsHTTPStatus(err error, statusCode int) bool {
	cE, ok := err.(*CustomError)

	if !ok {
		return false
	}

	return cE.StatusCode == statusCode
}

// IsErrorCode checks if the error is a `CustomError` with the
// specified code.
func IsErrorCode(err error, code string) bool {
	cE, ok := err.(*CustomError)

	if !ok {
		return false
	}

	return cE.Code == code
}

// IsRetryable checks if the error is a retryable `CustomError`.
func IsRetryable(err error) bool {
	cE, ok := err.(*CustomError)

	if !ok {
		return false
	}

	return cE.Retryable
}
