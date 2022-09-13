// It follows Rob Spike, and Dave Cheney design pattern for options.
//
// - Sensible defaults.
// - Highly configurable.
// - Allows anyone to easily implement their own options.
// - Can grow over time.
// - Self-documenting.
// - Safe for newcomers.
// - Never requires `nil` or an `empty` value to keep the compiler happy.
//
// SEE: https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html
// SEE: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis

package customerror

// Option allows to define error options.
type Option func(s *CustomError)

// Prepend options.
func prependOptions(source []Option, item Option) []Option {
	source = append(source, nil)

	copy(source[1:], source)

	source[0] = item

	return source
}

// WithError allows to specify an error which will be wrapped by the custom
// error.
func WithError(err error) Option {
	return func(s *CustomError) {
		s.Err = err
	}
}

// WithCode allows to specify an error code, such as "E1010".
func WithCode(code string) Option {
	return func(s *CustomError) {
		s.Code = code
	}
}

// WithStatusCode allows to specify the status code, such as "200".
func WithStatusCode(statucCode int) Option {
	return func(s *CustomError) {
		s.StatusCode = statucCode
	}
}
