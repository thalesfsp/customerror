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
	"sync"
	"testing"

	"github.com/emirpasic/gods/sets/treeset"
	"github.com/stretchr/testify/assert"
)

const (
	failedCreateSomethingMsg = "Failed to create something"
	code                     = "E1010"
	statusCode               = http.StatusNotFound
)

var (
	ErrFailedToReachServer        = errors.New("failed to reach servers")
	ErrFailedToReachServerDeep    = fmt.Errorf("%s. %w", ErrFailedToReachServer, errors.New("servers are broken"))
	ErrFailedToReachServerDeepRev = fmt.Errorf("%s. %w", errors.New("servers are broken"), ErrFailedToReachServer)
)

func TestNewLowLevel(t *testing.T) {
	type args struct {
		message string
		opts    []Option
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should work - with message",
			args: args{message: failedCreateSomethingMsg},
			want: failedCreateSomethingMsg,
		},
		{
			name: "should work - with message, and code",
			args: args{
				message: failedCreateSomethingMsg,
				opts:    []Option{WithCode(code)},
			},
			want: "E1010: Failed to create something",
		},
		{
			name: "should work - with message, and error",
			args: args{
				message: failedCreateSomethingMsg,
				opts:    []Option{WithError(ErrFailedToReachServer)},
			},
			want: "Failed to create something. Original Error: Failed to reach servers",
		},
		{
			name: "should work - with message, and deep error",
			args: args{
				message: failedCreateSomethingMsg,
				opts:    []Option{WithError(ErrFailedToReachServerDeep)},
			},
			want: "Failed to create something. Original Error: Failed to reach servers. Servers are broken",
		},
		{
			name: "should work - with message, and status code",
			args: args{
				message: failedCreateSomethingMsg,
				opts:    []Option{WithStatusCode(statusCode)},
			},
			want: "Failed to create something",
		},
		{
			name: "should work - with message, code, and error",
			args: args{
				message: failedCreateSomethingMsg,
				opts:    []Option{WithCode(code), WithError(ErrFailedToReachServer)},
			},
			want: "E1010: Failed to create something. Original Error: Failed to reach servers",
		},
		{
			name: "should work - with message, code, error, and deep error",
			args: args{
				message: failedCreateSomethingMsg,
				opts:    []Option{WithCode(code), WithError(ErrFailedToReachServerDeep)},
			},
			want: "E1010: Failed to create something. Original Error: Failed to reach servers. Servers are broken",
		},
		{
			name: "should work - with message, code, error, deep error, and status code",
			args: args{
				message: failedCreateSomethingMsg,
				opts:    []Option{WithCode(code), WithError(ErrFailedToReachServerDeep), WithStatusCode(statusCode)},
			},
			want: "E1010: Failed to create something. Original Error: Failed to reach servers. Servers are broken",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.args.message, tt.args.opts...)

			if !strings.EqualFold(got.Error(), tt.want) {
				t.Errorf("NewLowLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuiltin(t *testing.T) {
	ErrFailedToCreateFile := NewFailedToError("create file", WithCode("E1010"))
	ErrInvalidPath := NewInvalidError("path", WithCode("E1010"))
	ErrMissingPath := NewMissingError("path", WithCode("E1010"))
	ErrRequiredPath := NewRequiredError("path is", WithCode("E1010"))
	ErrNotFound := NewHTTPError(http.StatusNotFound, WithCode("E1010"))

	testFunc := func(e error) error { return e }

	type args struct {
		err error
	}
	tests := []struct {
		args               args
		expectedCode       string
		expectedStatusCode int
		name               string
		want               string
		wantAs             string
	}{
		{
			name:               "Should work - ErrFailedToCreateFile",
			expectedCode:       "E1010",
			expectedStatusCode: http.StatusInternalServerError,
			args: args{
				err: ErrFailedToCreateFile,
			},
			want:   "create file",
			wantAs: "create file",
		},
		{
			name:               "Should work - ErrInvalidPath",
			expectedCode:       "E1010",
			expectedStatusCode: http.StatusBadRequest,
			args: args{
				err: ErrInvalidPath,
			},
			want:   "invalid path",
			wantAs: "invalid path",
		},
		{
			name:               "Should work - ErrMissingPath",
			expectedCode:       "E1010",
			expectedStatusCode: http.StatusBadRequest,
			args: args{
				err: ErrMissingPath,
			},
			want:   "missing path",
			wantAs: "missing path",
		},
		{
			name:               "Should work - ErrRequiredPath",
			expectedCode:       "E1010",
			expectedStatusCode: http.StatusBadRequest,
			args: args{
				err: ErrRequiredPath,
			},
			want:   "path is required",
			wantAs: "path is required",
		},
		{
			name:               "Should work - ErrNotFound",
			expectedCode:       "E1010",
			expectedStatusCode: http.StatusNotFound,
			args: args{
				err: ErrNotFound,
			},
			want:   "not found",
			wantAs: "not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testFunc(tt.args.err)

			if !errors.Is(err, tt.args.err) {
				t.Errorf("Expected error to be (is) %v, got %v", tt.args.err, err)
			}

			errWrapped := fmt.Errorf("Wrapped %w", err)
			if !errors.Is(errWrapped, tt.args.err) {
				t.Errorf("Expected error to be (is - wrapped) %v, got %v", tt.args.err, errWrapped)
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf(`Expected message to be "%v", got "%v"`, tt.want, err)
			}

			var errAs *CustomError
			if errors.As(err, &errAs) {
				if !strings.Contains(errAs.Message, tt.wantAs) {
					t.Errorf(`Expected message to be (As)"%v", got "%v"`, tt.wantAs, errAs.Message)
				}

				if errAs.Code != tt.expectedCode {
					t.Errorf(`Expected code to be "%v", got "%v"`, tt.expectedCode, errAs.Code)
				}

				if errAs.StatusCode != tt.expectedStatusCode {
					t.Errorf(`Expected status code to be "%d", got "%d"`, tt.expectedStatusCode, errAs.StatusCode)
				}

				if errAs.APIError() == "" {
					t.Errorf(`Expected APIError to be empty, got "%v"`, errAs.APIError())
				}
			}
		})
	}
}

func TestCustomError_Unwrap(t *testing.T) {
	type fields struct {
		Code       string
		Err        error
		Message    string
		StatusCode int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    string
	}{
		{
			name: "Should work",
			fields: fields{
				Code:       "",
				Err:        errors.New("Wrapped error"),
				Message:    "Main error",
				StatusCode: 0,
			},
			wantErr: true,
			want:    "Wrapped error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cE := &CustomError{
				Code:       tt.fields.Code,
				Err:        tt.fields.Err,
				Message:    tt.fields.Message,
				StatusCode: tt.fields.StatusCode,
			}
			err := cE.Unwrap()

			if (err != nil) != tt.wantErr {
				t.Errorf("CustomError.Unwrap() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err.Error() != tt.want {
				t.Errorf("CustomError.Unwrap() message = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestNew_deepNestedErrors(t *testing.T) {
	expectedErrMsg := "custom message. Original Error: layer 3. layer 2. layer 1"

	layer1 := errors.New("layer 1")

	layer2 := fmt.Errorf("layer 2. %w", layer1)

	layer3 := fmt.Errorf("layer 3. %w", layer2)

	ErrLayered := New("custom message", WithError(layer3))
	if ErrLayered.Error() != expectedErrMsg {
		t.Errorf("CustomError deep nested errors got %s, want %s", ErrLayered, expectedErrMsg)
	}

	testFunc := func() error { return ErrLayered }

	errLayered := testFunc()

	if !errors.Is(errLayered, ErrLayered) {
		t.Errorf("Expected %v be ErrLayered", errLayered)
	}

	errSome := errors.New("Some error")

	errWrapped := Wrap(errLayered, errSome)

	if !errors.Is(errWrapped, ErrLayered) {
		t.Errorf("Expected %v be ErrLayered", errWrapped)
	}

	expectedErrWrappedMsg := "custom message. Original Error: layer 3. layer 2. layer 1. Wrapped Error(s): Some error"

	if errWrapped.Error() != expectedErrWrappedMsg {
		t.Errorf("Expected %v to be %s", errWrapped.Error(), expectedErrWrappedMsg)
	}
}

func TestWrap(t *testing.T) {
	expectedErrMsg := "custom message. Original Error: layer 3. layer 2. layer 1"

	layer1 := errors.New("layer 1")

	layer2 := fmt.Errorf("layer 2. %w", layer1)

	layer3 := fmt.Errorf("layer 3. %w", layer2)

	ErrLayered := New("custom message", WithError(layer3))
	if ErrLayered.Error() != expectedErrMsg {
		t.Errorf("Wrap got %s, want %s", ErrLayered, expectedErrMsg)
	}

	testFunc := func() error { return ErrLayered }

	errLayered := testFunc()

	if !errors.Is(errLayered, ErrLayered) {
		t.Errorf("Wrap Is got %s, want %s", errLayered, ErrLayered)
	}

	errSome := errors.New("Some error")

	if !errors.Is(Wrap(errLayered, errSome), ErrLayered) {
		t.Errorf("Wrap Is got %s, want %s", errSome, ErrLayered)
	}
}

// Test invalid custom error.
//
//nolint:forcetypeassert
func TestNew_InvalidCustomError(t *testing.T) {
	t.Setenv("CUSTOMERROR_ENVIRONMENT", "testing")

	// Recover from panic, and test error message.
	defer func() {
		if r := recover(); r != nil {
			if !strings.Contains(r.(string), "Invalid custom error") {
				t.Fatal("Expected panic message to be 'hahaha', got", r)
			}
		}
	}()

	_ = New("")
}

func Test_CustomError_MarshalJSON(t *testing.T) {
	fields := sync.Map{}
	fields.Store("field1", "value1")
	fields.Store("field2", 2)

	tests := []struct {
		name     string
		cE       *CustomError
		expected string
	}{
		{
			name: "with all fields",
			cE: &CustomError{
				Code:       "E1010",
				Err:        errors.New("Some error"),
				Fields:     &fields,
				Message:    "An error occurred",
				StatusCode: http.StatusBadRequest,
				Tags:       &Set{treeset.NewWithStringComparator("tag1", "tag2")},
				ignore:     false,
			},
			expected: `{"code":"E1010","field1":"value1","field2":2,"message":"An error occurred. Original Error: Some error","tags":["tag1","tag2"]}`,
		},
		{
			name: "with message only",
			cE: &CustomError{
				Message: "An error occurred",
			},
			expected: `{"message":"An error occurred"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.cE)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(b))
		})
	}
}
