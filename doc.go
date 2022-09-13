// Copyright 2021 The customerror Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Package customerror provides the base block to create custom errors. It also
// provides built-in custom errors covering some common cases. A Custom Error
// provides context - a `Message` to an optionally wrapped `Err`. Additionally a
// `Code` - for example "E1010", and `StatusCode` can be provided. Both static
// (pre-created), and dynamic (in-line) errors can be easily created. `Code`
// helps a company build a catalog of errors, which helps, and improves customer
// service.
//
// Examples:
//
// See `example_test.go` or the Example section of the GoDoc documention.
package customerror
