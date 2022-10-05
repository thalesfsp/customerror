// Copyright 2021 The customerror Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package customerror

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

// CustomError is the base block to create custom errors. It provides context -
// a `Message` to an optional `Err`. Additionally a `Code` - for example "E1010",
// and `StatusCode` can be provided.
type CustomError struct {
	// Code can be any custom code, e.g.: E1010.
	Code string `json:"code,omitempty" validate:"omitempty,startswith=E,gte=2"`

	// Err optionally wraps the original error.
	Err error `json:"-"`

	// Human readable message. Minimum length: 3.
	Message string `json:"message" validate:"required,gte=3"`

	// StatusCode is a valid HTTP status code, e.g.: 404.
	StatusCode int `json:"-" validate:"omitempty,gte=100,lte=511"`
}

//////
// Error interface implementation.
//////

// Error interface implementation returns the properly formatted error message.
func (cE *CustomError) Error() string {
	errMsg := cE.Message

	if cE.Code != "" {
		errMsg = fmt.Sprintf("%s: %s", cE.Code, errMsg)
	}

	if cE.Err != nil {
		errMsg = fmt.Errorf("%s. Original Error: %w", errMsg, cE.Err).Error()
	}

	return errMsg
}

// APIError is like error plus status code information.
func (cE *CustomError) APIError() string {
	errMsg := cE.Message

	if cE.Code != "" {
		errMsg = fmt.Sprintf("%s: %s", cE.Code, errMsg)
	}

	if cE.StatusCode != 0 {
		errMsg = fmt.Sprintf("%s (%d - %s)", errMsg, cE.StatusCode, http.StatusText(cE.StatusCode))
	}

	if cE.Err != nil {
		errMsg = fmt.Errorf("%s. Original Error: %w", errMsg, cE.Err).Error()
	}

	return errMsg
}

// Unwrap interface implementation returns inner error.
func (cE *CustomError) Unwrap() error {
	return cE.Err
}

// Is interface implementation ensures chain continuity. Treats `CustomError` as
// equivalent to `err`.
//
//nolint:errorlint
func (cE *CustomError) Is(err error) bool {
	return cE.Err == err
}

// MarshalJSON implements the json.Marshaler interface.
//
// See: https://gist.github.com/thalesfsp/3a1252530750e2370345a2418721ff54
func (cE *CustomError) MarshalJSON() ([]byte, error) {
	type Alias CustomError

	b := &struct {
		*Alias
	}{
		Alias: (*Alias)(cE),
	}

	b.Message = cE.Error()

	return json.Marshal(b)
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

//////
// Factory.
//////

// New is the custom error factory.
func New(message string, opts ...Option) error {
	cE := &CustomError{
		Message: message,
	}

	for _, opt := range opts {
		opt(cE)
	}

	if err := validator.New().Struct(cE); err != nil {
		log.Fatalf("Invalid custom error. %s\n", err)

		return nil
	}

	return cE
}
