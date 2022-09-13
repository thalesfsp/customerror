# customerror

`customerror` provides the base block to create custom errors. It also provides built-in custom errors covering some common cases. A Custom Error provides context - a `Message` to an optionally wrapped `Err`. Additionally a `Code` - for example "E1010", and `StatusCode` can be provided. Both static (pre-created), and dynamic (in-line) errors can be easily created. `Code` helps a company build a catalog of errors, which helps, and improves customer service.

## Install

`$ go get github.com/thalesfsp/customerror@vX.Y.Z`

## Usage

See [`example_test.go`](example_test.go), and [`customerror_test.go`](customerror_test.go) file.

## Documentation

Run `$ make doc` or check out [online](https://pkg.go.dev/github.com/thalesfsp/customerror).

## Development

Check out [CONTRIBUTION](CONTRIBUTION.md).

### Release

1. Update [CHANGELOG](CHANGELOG.md) accordingly.
2. Once changes from MR are merged.
3. Tag and release.

## Roadmap

Check out [CHANGELOG](CHANGELOG.md).
