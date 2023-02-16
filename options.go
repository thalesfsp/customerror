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

import "strings"

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
	return func(cE *CustomError) {
		cE.Err = err
	}
}

// WithCode allows to specify an error code, such as "E1010".
func WithCode(code string) Option {
	return func(cE *CustomError) {
		cE.Code = code
	}
}

// WithStatusCode allows to specify the status code, such as "200".
func WithStatusCode(statusCode int) Option {
	return func(cE *CustomError) {
		cE.StatusCode = statusCode
	}
}

// WithIgnoreFunc ignores an error if the specified function returns true.
func WithIgnoreFunc(f func(cE *CustomError) bool) Option {
	return func(cE *CustomError) {
		if f(cE) {
			cE.ignore = true
		}
	}
}

// WithIgnoreString ignores an error if the error message, or the the underlying
// error message contains the specified string.
func WithIgnoreString(s ...string) Option {
	return WithIgnoreFunc(func(cE *CustomError) bool {
		for _, str := range s {
			if strings.Contains(cE.Message, str) {
				return true
			}

			if cE.Err != nil && strings.Contains(cE.Err.Error(), str) {
				return true
			}
		}

		return false
	})
}

// WithTag allows to specify tags for the error.
func WithTag(tag ...string) Option {
	return func(cE *CustomError) {
		cE.Tags = tag
	}
}

// WithField allows to specify fields for the error.
func WithField(fields map[string]interface{}) Option {
	return func(cE *CustomError) {
		cE.Fields = fields
	}
}
