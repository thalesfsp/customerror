# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.16] - 2023-03-02
### Changed
- Fixed `fields` `MarshalJSON`.

## [1.0.15] - 2023-03-02
### Added
- `WithField` which allows to pass a single Key/Value pair. `WithFields` which allows to set the whole map of fields. It's thread safe.

## [1.0.14] - 2023-02-16
### Added
- Added the ability to factorize (`NewFactory`) custom errors with pre-defined tags and fields.

## [1.0.13] - 2023-02-16
### Added
- Added the ability to add fields (`WithFields`) to the error enhancing the error message with structured information.

## [1.0.12] - 2022-10-21
### Added
- `WithIgnoreString` now also check the underlying error message for the given string.

## [1.0.11] - 2022-10-21
### Added
- Added the ability to categorize errors with a custom tags (`WithTags`) enhancing the error message tags.

## [1.0.10] - 2022-10-21
### Changed
- Fixed type in the `WithStatusCode` `Option`.

## [1.0.9] - 2022-10-21
### Added
- `WithIgnoreString` `Option` accepts a slice of strings.
- Added ability to create messages with just `StatusCode`, or just `Code`, but no `Message`. `StatusCode` takes precedence over `Code`.

## [1.0.8] - 2022-10-21
### Added
- `WithIgnoreFunc` `Option` which allows to ignore the error (return `nil`) under the specified condition.
- `WithIgnoreString` `Option` which allows to ignore the error (return `nil`) if `Message` contains the specified **string**.
- `NewHTTPError` test.

## [1.0.7] - 2022-10-20
### Added
- `NewHTTPError` for simple HTTP error messages, e.g.: "not found".

## [1.0.6] - 2022-10-04
### Changed
- Improved JSON representation/marshalling
- Added tests

## [1.0.5] - 2022-09-13
### Changed
- Updated deps

## [1.0.4] - 2022-08-08
### Added
- Updating github CI dependencies
- Updating CI to use Go 1.19
- Updating CI linter version
- Linting the code

## [1.0.3] - 2022-02-18
### Added
- Tests covering `.Is`.`

### Changed
- `.Is` is on a pointer instead of value.

## [1.0.2] - 2022-02-18
### Added
- API info is only added to error message on `APIError`.

## [1.0.1] - 2022-02-18
### Added
- Added ability to pre-append `Option`s.

## [1.0.0] - 2022-02-17
### Added
- Functional `Option`s.

### Changed
- `New` now implements the functional optional pattern.

### Removed
- Removed `SetStatusCode`.

## [0.0.2] - 2021-09-27
### Changed
- `Wrap` now accepts a list of errors.

## [0.0.1] - 2021-09-24
### Checklist
- [x] CI Pipeline:
  - [x] Lint
  - [x] Tests
- [x] Documentation:
  - [x] Package's documentation (`doc.go`)
  - [x] Meaningful code comments, and symbol names (`const`, `var`, `func`)
  - [x] `GoDoc` server tested
  - [x] `README.md`
  - [x] `LICENSE`
    - [x] Files has LICENSE in the header
  - [x] Useful `CHANGELOG.md`
  - [x] Clear `CONTRIBUTION.md`
- Automation:
  - [x] `Makefile`
- Testing:
  - [x] Coverage 80%+
  - [x] Unit test
  - [x] Real testing
- Examples:
  - [x] Example's test file

### Added
- [x] Ability to create custom errors.
- [x] Ability to create custom errors with code.
- [x] Ability to create custom errors with status code.
- [x] Ability to create custom errors with message.
- [x] Ability to create custom errors wrapping an error.
- [x] Ability to create static (pre-created) custom errors.
- [x] Ability to create dynamic (in-line) custom errors.
- [x] Ability to print a custom error with a dynamic, and custom message.
