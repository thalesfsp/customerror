// Copyright 2021 The customerror Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package customerror

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/emirpasic/gods/sets/treeset"
	"github.com/go-playground/validator/v10"
)

//////
// Consts, vars, and types.
//////

// validate is the shared, reusable validator instance. The validator caches
// struct metadata internally, so a single instance is meant to be reused
// across calls (and is safe for concurrent use).
var validate = validator.New()

// reservedJSONKeys are the top-level keys owned by the CustomError structure
// when marshaling to JSON. User-provided fields must not be allowed to clobber
// them.
var reservedJSONKeys = map[string]struct{}{
	"message":    {},
	"code":       {},
	"tags":       {},
	"retryable":  {},
	"retried":    {},
	"statusCode": {},
}

//////
// Helpers.
//////

// Copy performs a deep copy of the src CustomError into the target CustomError.
// It ensures that all fields, including nested structures like maps and sets,
// are properly copied. This function is useful when you need to create a new
// CustomError instance based on an existing one, while avoiding any shared
// references to mutable fields.
//
// A nil `src` is a no-op (target is returned unchanged); a nil `target` is
// replaced by a fresh CustomError so the call never panics.
func Copy(src, target *CustomError) *CustomError {
	if target == nil {
		target = &CustomError{}
	}

	if src == nil {
		return target
	}

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
	//
	// NOTE: When src and target share the very same Set (same pointer) there
	// is nothing to merge - and it must be skipped so the target's write lock
	// is never requested while iterating the (same) source.
	if src.Tags != nil && src.Tags != target.Tags {
		if target.Tags == nil {
			target.Tags = newSet()
		}

		// Snapshot the source values first so no lock is held on the source
		// while the target is being mutated.
		for _, value := range src.Tags.Values() {
			target.Tags.Add(value)
		}
	}

	return target
}

// Process fields and add them to the error message. If `fields` is nil or
// empty, the message is returned unchanged (no dangling ". Fields:"). Fields
// are sorted by key so the output is deterministic.
func processFields(
	errMsg string,
	fields *sync.Map,
) string {
	if fields == nil {
		return errMsg
	}

	pairs := []string{}

	fields.Range(func(k, v interface{}) bool {
		pairs = append(pairs, fmt.Sprintf("%v=%v", k, v))

		return true
	})

	if len(pairs) == 0 {
		return errMsg
	}

	sort.Strings(pairs)

	return fmt.Sprintf("%s. Fields: %s", errMsg, strings.Join(pairs, ", "))
}

// addUserFieldsToJSON copies user-provided fields into the JSON map `temp`,
// skipping empty keys, nil values, and any key reserved by the CustomError
// structure (so user fields can never clobber structural JSON keys).
func addUserFieldsToJSON(temp map[string]interface{}, fields *sync.Map) {
	if fields == nil {
		return
	}

	fields.Range(func(k, v interface{}) bool {
		key, ok := k.(string)
		if !ok || key == "" || v == nil {
			return true
		}

		if _, reserved := reservedJSONKeys[key]; reserved {
			return true
		}

		temp[key] = v

		return true
	})
}

// Set is a wrapper around the treeset.Set, providing a collection
// that stores unique elements in a sorted order. It is used in the
// CustomError struct to maintain a sorted set of tags.
//
// The methods defined on Set itself - `Add`, `Remove`, `Contains`, `Empty`,
// `Size`, `Clear`, `Values`, `Each`, `String`, and `MarshalJSON` - are safe
// for concurrent use by multiple goroutines. Calling any other method of the
// embedded treeset.Set directly (e.g. `Iterator`, `UnmarshalJSON`) bypasses
// the lock and is NOT safe for concurrent use.
type Set struct {
	*treeset.Set

	// mu guards the embedded treeset.Set.
	mu sync.RWMutex
}

// newSet creates a new Set, optionally initialized with the given values.
func newSet(values ...interface{}) *Set {
	return &Set{Set: treeset.NewWithStringComparator(values...)}
}

// Add adds the given items to the set. Safe for concurrent use.
func (s *Set) Add(items ...interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Set.Add(items...)
}

// Remove removes the given items from the set. Safe for concurrent use.
func (s *Set) Remove(items ...interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Set.Remove(items...)
}

// Contains checks whether the set contains all the given items. Safe for
// concurrent use.
func (s *Set) Contains(items ...interface{}) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Set.Contains(items...)
}

// Empty checks whether the set has no elements. Safe for concurrent use.
func (s *Set) Empty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Set.Empty()
}

// Size returns the number of elements in the set. Safe for concurrent use.
func (s *Set) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Set.Size()
}

// Clear removes all elements from the set. Safe for concurrent use.
func (s *Set) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Set.Clear()
}

// Values returns all elements in the set, sorted. Safe for concurrent use.
func (s *Set) Values() []interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Set.Values()
}

// Each calls the given function once for each element, in sorted order. Safe
// for concurrent use.
//
// NOTE: `f` must NOT call methods of the same Set (the lock is held for the
// whole iteration); snapshot with `Values` first if that is needed.
func (s *Set) Each(f func(index int, value interface{})) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.Set.Each(f)
}

// String implements the Stringer interface for the Set type.
// It returns a comma-separated string representation of all elements
// in the set, useful for debugging and error message formatting. Safe for
// concurrent use.
func (s *Set) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := []string{}

	s.Set.Each(func(_ int, value interface{}) {
		items = append(items, fmt.Sprintf("%v", value))
	})

	return strings.Join(items, ", ")
}

// MarshalJSON implements the json.Marshaler interface, delegating to the
// embedded treeset.Set implementation. Safe for concurrent use.
func (s *Set) MarshalJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Set.MarshalJSON()
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

	if cE.Tags != nil && !cE.Tags.Empty() {
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
	// Preserve the original nil semantics.
	if cE.Err == nil || err == nil {
		return cE.Err == err
	}

	// Guard against panics when comparing non-comparable error values (e.g. an
	// error whose dynamic type contains a slice or a map).
	if !reflect.TypeOf(cE.Err).Comparable() || !reflect.TypeOf(err).Comparable() {
		return false
	}

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

	// Add user fields, skipping any that would clobber the structural keys.
	addUserFieldsToJSON(temp, cE.Fields)

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

	if cE.Tags != nil && !cE.Tags.Empty() {
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
		// Get the template by the language.
		template, err := GetTemplate(string(finalCE.language), errorType)
		if err != nil {
			// Fall back to the root language (e.g. "pt-BR" -> "pt").
			rootTemplate, rootErr := GetTemplate(finalCE.language.GetRoot(), errorType)
			if rootErr != nil {
				// No template available for this language/error type. Degrade
				// gracefully by returning the (unprefixed) message instead of
				// panicking.
				return finalCE
			}

			template = rootTemplate
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
	if finalCE == nil {
		return nil
	}

	// Honor WithIgnoreFunc/WithIgnoreString consistently.
	if finalCE.ignore {
		return nil
	}

	if finalCE.language == "" {
		// The prefixed message may itself trigger an ignore option, in which
		// case the inner constructor returns nil; guard the type assertion.
		inner := NewFailedToError(finalCE.Message, opts...)
		if inner == nil {
			return nil
		}

		finalCE = Copy(inner.(*CustomError), finalCE)

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
	if finalCE == nil {
		return nil
	}

	// Honor WithIgnoreFunc/WithIgnoreString consistently.
	if finalCE.ignore {
		return nil
	}

	if finalCE.language == "" {
		// The prefixed message may itself trigger an ignore option, in which
		// case the inner constructor returns nil; guard the type assertion.
		inner := NewInvalidError(finalCE.Message, opts...)
		if inner == nil {
			return nil
		}

		finalCE = Copy(inner.(*CustomError), finalCE)

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
	if finalCE == nil {
		return nil
	}

	// Honor WithIgnoreFunc/WithIgnoreString consistently.
	if finalCE.ignore {
		return nil
	}

	if finalCE.language == "" {
		// The prefixed message may itself trigger an ignore option, in which
		// case the inner constructor returns nil; guard the type assertion.
		inner := NewMissingError(finalCE.Message, opts...)
		if inner == nil {
			return nil
		}

		finalCE = Copy(inner.(*CustomError), finalCE)

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
	if finalCE == nil {
		return nil
	}

	// Honor WithIgnoreFunc/WithIgnoreString consistently.
	if finalCE.ignore {
		return nil
	}

	if finalCE.language == "" {
		// The prefixed message may itself trigger an ignore option, in which
		// case the inner constructor returns nil; guard the type assertion.
		inner := NewRequiredError(finalCE.Message, opts...)
		if inner == nil {
			return nil
		}

		finalCE = Copy(inner.(*CustomError), finalCE)

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

	// Work on a copy so the receiver (often a reusable factory/catalog error)
	// is never mutated.
	finalCE := Copy(cE, &CustomError{})

	// Use the receiver's status code if it was explicitly set, otherwise fall
	// back to the provided one.
	if finalCE.StatusCode == 0 {
		finalCE.StatusCode = statusCode
	}

	httpErr := NewHTTPError(finalCE.StatusCode, opts...)
	if httpErr == nil {
		// Ignored via WithIgnoreFunc/WithIgnoreString.
		return nil
	}

	httpCE := httpErr.(*CustomError)

	finalErrorMessage := httpCE.Message

	// Apply options.
	for _, opt := range opts {
		opt(finalCE)
	}

	if finalCE.ignore {
		return nil
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

	// Honor WithIgnoreFunc/WithIgnoreString and guard the type assertion in
	// case the inner constructor ignored the error (returned nil).
	if finalCE.ignore {
		return nil
	}

	inner := New(finalCE.Message, opts...)
	if inner == nil {
		return nil
	}

	finalCE = Copy(inner.(*CustomError), finalCE)

	return finalCE
}

//////
// Exported functionalities.
//////

// wrappedError is the error type returned by Wrap. It preserves the identity of
// every wrapped error (so errors.Is/errors.As work for all of them) while
// keeping a human-readable, stable message format.
type wrappedError struct {
	// customError is the primary error being wrapped.
	customError error

	// errs are the additional errors wrapped around customError.
	errs []error
}

// Error implements the error interface, preserving the message format:
// "<customError>. Wrapped Error(s): <e1>. <e2>...".
func (w *wrappedError) Error() string {
	errMsgs := make([]string, 0, len(w.errs))

	for _, err := range w.errs {
		errMsgs = append(errMsgs, err.Error())
	}

	return fmt.Sprintf("%s. Wrapped Error(s): %s", w.customError.Error(), strings.Join(errMsgs, ". "))
}

// Unwrap returns all wrapped errors, enabling errors.Is/errors.As to match any
// of them (Go 1.20+ multi-error unwrapping).
func (w *wrappedError) Unwrap() []error {
	all := make([]error, 0, len(w.errs)+1)

	all = append(all, w.customError)
	all = append(all, w.errs...)

	return all
}

// Wrap wraps `customError` together with the additional `errors`. The returned
// error preserves the identity of every wrapped error, so errors.Is and
// errors.As match against `customError` and each of `errors`. Nil errors are
// ignored: if `customError` is nil the first non-nil additional error takes
// its place, if every error is nil, nil is returned, and if there is nothing
// to wrap the single remaining error is returned as-is (no dangling
// "Wrapped Error(s)" suffix).
func Wrap(customError error, errors ...error) error {
	nonNil := make([]error, 0, len(errors))

	for _, err := range errors {
		if err != nil {
			nonNil = append(nonNil, err)
		}
	}

	if customError == nil {
		if len(nonNil) == 0 {
			return nil
		}

		customError, nonNil = nonNil[0], nonNil[1:]
	}

	if len(nonNil) == 0 {
		return customError
	}

	return &wrappedError{
		customError: customError,
		errs:        nonNil,
	}
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

	return cE
}

// New creates a new validated custom error returning it as en `error`.
//
// NOTE: Creating an error with invalid attributes (for example, an empty
// message) is a programming error. In that case New panics (recoverable)
// rather than terminating the host process - a library must never call
// os.Exit on its caller's behalf.
func New(message string, opts ...Option) error {
	cE := newInternal(prependOptions(opts, WithMessage(message))...)

	// Should be able to programatically ignore errors (`WithIgnoreFunc`).
	if cE.ignore {
		return nil
	}

	if err := validate.Struct(cE); err != nil {
		log.Panicf("Invalid custom error. %s\n", err)
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
	cE := newInternal(prependOptions(opts, WithMessage(message))...)

	// Should be able to programatically ignore errors (`WithIgnoreFunc`).
	if cE.ignore {
		return nil
	}

	return cE
}

// IsCustomError checks if the error is - or wraps - a `CustomError`
// (it traverses the error chain, see errors.As).
func IsCustomError(err error) bool {
	var cE *CustomError

	return errors.As(err, &cE)
}

// To converts the error to a `CustomError`. It traverses the error chain
// (see errors.As), so a `CustomError` wrapped by fmt.Errorf ("%w") or `Wrap`
// is found too; the first `CustomError` in the chain is returned.
func To(err error) (*CustomError, bool) {
	var cE *CustomError

	if !errors.As(err, &cE) {
		return nil, false
	}

	return cE, true
}

// From returns a copy of `err` with the given options applied. If `err` is a
// `CustomError`, the original is left untouched (not mutated) and a modified
// copy is returned - this makes it safe to use with shared sentinel errors. If
// `err` isn't a custom error, a new custom error wrapping it (with the given
// options) is returned.
//
// NOTE: Unlike `To`, `From` intentionally only treats a DIRECT `*CustomError`
// specially - a custom error nested inside another error is wrapped like any
// other error, so no outer context is ever dropped.
//
//nolint:errorlint
func From(err error, opts ...Option) error {
	if cE, ok := err.(*CustomError); ok {
		// Operate on a copy so the original error (which may be a shared
		// sentinel) is never mutated.
		finalCE := Copy(cE, &CustomError{})

		for _, opt := range opts {
			opt(finalCE)
		}

		return finalCE
	}

	// WithError properly deals with Golang errors (unwrapping, etc).
	opts = append(opts, WithError(err))

	return New(err.Error(), opts...)
}

// IsHTTPStatus checks if the error is - or wraps - a `CustomError` with the
// specified HTTP status code (the first `CustomError` in the chain is
// checked, see errors.As).
func IsHTTPStatus(err error, statusCode int) bool {
	cE, ok := To(err)

	if !ok {
		return false
	}

	return cE.StatusCode == statusCode
}

// IsErrorCode checks if the error is - or wraps - a `CustomError` with the
// specified code (the first `CustomError` in the chain is checked, see
// errors.As).
func IsErrorCode(err error, code string) bool {
	cE, ok := To(err)

	if !ok {
		return false
	}

	return cE.Code == code
}

// IsRetryable checks if the error is - or wraps - a retryable `CustomError`
// (the first `CustomError` in the chain is checked, see errors.As).
func IsRetryable(err error) bool {
	cE, ok := To(err)

	if !ok {
		return false
	}

	return cE.Retryable
}
