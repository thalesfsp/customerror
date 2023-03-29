// Copyright 2021 The customerror Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package customerror

import (
	"fmt"
	"net/http"
	"strings"
)

//////
// Built-in.
//////

// NewFailedToError is the building block for errors usually thrown when some
// action failed, e.g: "Failed to create host". Default status code is `500`.
//
// NOTE: Status code can be redefined, call `SetStatusCode`.
func NewFailedToError(message string, opts ...Option) error {
	return New(fmt.Sprintf("failed to %s", message), prependOptions(
		opts,
		WithStatusCode(http.StatusInternalServerError),
	)...)
}

// NewInvalidError is the building block for errors usually thrown when
// something fail validation, e.g: "Invalid port". Default status code is `400`.
//
// NOTE: Status code can be redefined, call `SetStatusCode`.
func NewInvalidError(message string, opts ...Option) error {
	return New(fmt.Sprintf("invalid %s", message), prependOptions(
		opts,
		WithStatusCode(http.StatusBadRequest),
	)...)
}

// NewMissingError is the building block for errors usually thrown when required
// information is missing, e.g: "Missing host". Default status code is `400`.
//
// NOTE: Status code can be redefined, call `SetStatusCode`.
func NewMissingError(message string, opts ...Option) error {
	return New(fmt.Sprintf("missing %s", message), prependOptions(
		opts,
		WithStatusCode(http.StatusBadRequest),
	)...)
}

// NewRequiredError is the building block for errors usually thrown when
// required information is missing, e.g: "Port is required". Default status code
// is `400`.
//
// NOTE: Status code can be redefined, call `SetStatusCode`.
func NewRequiredError(message string, opts ...Option) error {
	return New(fmt.Sprintf("%s required", message), prependOptions(
		opts,
		WithStatusCode(http.StatusBadRequest),
	)...)
}

// NewNotFoundError is the building block for errors usually thrown when something
// is not found, e.g: "Host not found". Default status code is `400`.
//
// NOTE: Status code can be redefined, call `SetStatusCode`.
func NewNotFoundError(message string, opts ...Option) error {
	return New(fmt.Sprintf("%s not found", message), prependOptions(
		opts,
		WithStatusCode(http.StatusBadRequest),
	)...)
}

// NewHTTPError is the building block for simple HTTP errors, e.g.: Not Found.
func NewHTTPError(statusCode int, opts ...Option) error {
	return New(strings.ToLower(http.StatusText(statusCode)), prependOptions(
		opts,
		WithStatusCode(statusCode),
	)...)
}
