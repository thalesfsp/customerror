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

	"github.com/go-playground/validator/v10"
)

//////
// Helpers.
//////

// Process fields and add them to the error message.
func processFields(errMsg string, fields *sync.Map) string {
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

// dedupTags removes duplicate tags.
func dedupTags(tags []string) []string {
	keys := make(map[string]bool)

	list := []string{}

	for _, entry := range tags {
		if _, value := keys[entry]; !value {
			keys[entry] = true

			list = append(list, entry)
		}
	}

	return list
}

// CustomError is the base block to create custom errors. It provides context -
// a `Message` to an optional `Err`. Additionally a `Code` - for example "E1010",
// and `StatusCode` can be provided.
type CustomError struct {
	// Code can be any custom code, e.g.: E1010.
	Code string `json:"code,omitempty" validate:"omitempty,startswith=E,gte=2"`

	// Err optionally wraps the original error.
	Err error `json:"-"`

	// Field enhances the error message with more structured information.
	Fields *sync.Map `json:"fields,omitempty"`

	// Human readable message. Minimum length: 3.
	Message string `json:"message" validate:"required,gte=3"`

	// StatusCode is a valid HTTP status code, e.g.: 404.
	StatusCode int `json:"-" validate:"omitempty,gte=100,lte=511"`

	// Tags is a list of tags which helps to categorize the error.
	Tags []string `json:"tags,omitempty"`

	// If set to true, the error will be ignored (return nil).
	ignore bool `json:"-"`
}

//////
// Error interface implementation.
//////

// JustError returns the error message without any additional information.
func (cE *CustomError) JustError() string {
	errMsg := cE.Message

	if cE.Err != nil {
		errMsg = fmt.Errorf("%s. Original Error: %w", errMsg, cE.Err).Error()
	}

	return errMsg
}

// Error interface implementation returns the properly formatted error message.
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
		errMsg = fmt.Sprintf("%s. Tags: %s", errMsg, strings.Join(cE.Tags, ", "))
	}

	errMsg = processFields(errMsg, cE.Fields)

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
		errMsg = fmt.Sprintf("%s. Tags: %s", errMsg, strings.Join(cE.Tags, ", "))
	}

	errMsg = processFields(errMsg, cE.Fields)

	return errMsg
}

// Unwrap interface implementation returns inner error.
func (cE *CustomError) Unwrap() error {
	return cE.Err
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

// MarshalJSON implements the json.Marshaler interface.
//
// SEE https://gist.github.com/thalesfsp/3a1252530750e2370345a2418721ff54
func (cE *CustomError) MarshalJSON() ([]byte, error) {
	// Define a temporary map that matches the desired JSON format.
	temp := make(map[string]interface{})

	// Populate the temporary map.
	temp["code"] = cE.Code
	temp["message"] = cE.JustError()
	temp["tags"] = cE.Tags

	// Convert the sync.Map to a regular map so that we can iterate over its keys.
	fields := syncMapToMap(cE.Fields)

	// Populate the fields of the temporary map.
	if len(fields) > 0 {
		for k, v := range fields {
			temp[k] = v
		}
	}

	// Serialize the temporary map to JSON.
	return json.Marshal(temp)
}

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

// NewFailedToError is the building block for errors usually thrown when some
// action failed, e.g: "Failed to create host". Default status code is `500`.
//
// NOTE: Status code can be redefined, call `SetStatusCode`.
func (cE *CustomError) NewFailedToError(message string, opts ...Option) error {
	finalOpts := []Option{
		WithTag(cE.Tags...),
		WithFields(syncMapToMap(cE.Fields)),
	}

	// Add opts to finalOpts.
	finalOpts = append(finalOpts, opts...)

	return NewFailedToError(message, finalOpts...)
}

// NewInvalidError is the building block for errors usually thrown when
// something fail validation, e.g: "Invalid port". Default status code is `400`.
//
// NOTE: Status code can be redefined, call `SetStatusCode`.
func (cE *CustomError) NewInvalidError(message string, opts ...Option) error {
	finalOpts := []Option{
		WithTag(cE.Tags...),
		WithFields(syncMapToMap(cE.Fields)),
	}

	// Add opts to finalOpts.
	finalOpts = append(finalOpts, opts...)

	return NewInvalidError(message, finalOpts...)
}

// NewMissingError is the building block for errors usually thrown when required
// information is missing, e.g: "Missing host". Default status code is `400`.
//
// NOTE: Status code can be redefined, call `SetStatusCode`.
func (cE *CustomError) NewMissingError(message string, opts ...Option) error {
	finalOpts := []Option{
		WithTag(cE.Tags...),
		WithFields(syncMapToMap(cE.Fields)),
	}

	// Add opts to finalOpts.
	finalOpts = append(finalOpts, opts...)

	return NewMissingError(message, finalOpts...)
}

// NewRequiredError is the building block for errors usually thrown when
// required information is missing, e.g: "Port is required". Default status code is `400`.
//
// NOTE: Status code can be redefined, call `SetStatusCode`.
func (cE *CustomError) NewRequiredError(message string, opts ...Option) error {
	finalOpts := []Option{
		WithTag(cE.Tags...),
		WithFields(syncMapToMap(cE.Fields)),
	}

	// Add opts to finalOpts.
	finalOpts = append(finalOpts, opts...)

	return NewRequiredError(message, finalOpts...)
}

// NewHTTPError is the building block for simple HTTP errors, e.g.: Not Found.
func (cE *CustomError) NewHTTPError(statusCode int, opts ...Option) error {
	finalOpts := []Option{
		WithTag(cE.Tags...),
		WithFields(syncMapToMap(cE.Fields)),
	}

	// Add opts to finalOpts.
	finalOpts = append(finalOpts, opts...)

	return NewHTTPError(statusCode, finalOpts...)
}

// NewChildError creates a new `CustomError` with the same fields and tags of
// the parent `CustomError` plus the new fields and tags passed as arguments.
func (cE *CustomError) NewChildError(fields map[string]interface{}, tags ...string) *CustomError {
	childCE := &CustomError{
		Fields: cE.Fields,
		Tags:   cE.Tags,
	}

	// Merge the fields to cE.Fields.
	for k, v := range fields {
		childCE.Fields.Store(k, v)
	}

	// Merge the tags to cE.Tags.
	childCE.Tags = append(childCE.Tags, tags...)

	// Remove duplicates.
	childCE.Tags = dedupTags(childCE.Tags)

	return childCE
}

//////
// Factory.
//////

// New is the custom error factory.
func New(message string, opts ...Option) error {
	cE := &CustomError{
		Message: message,
		ignore:  false,
	}

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

// NewFactory returns a new custom error with pre-defined fields and tags. It
// can then be used to generate other custom errors such as:
// - `NewFailedToError`
// - `NewInvalidError`
// - `NewMissingError`
// - `NewRequiredError`
// - `NewHTTPError`.
func NewFactory(fields map[string]interface{}, tags ...string) *CustomError {
	return &CustomError{
		Fields: mapToSyncMap(fields),
		Tags:   tags,
	}
}
