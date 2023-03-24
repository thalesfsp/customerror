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

import (
	"strings"
	"sync"

	"github.com/emirpasic/gods/sets/treeset"
)

//////
// Consts, vars, and types.
//////

// Option allows to define error options.
type Option func(s *CustomError)

// Prepend options.
func prependOptions(source []Option, item Option) []Option {
	source = append(source, nil)

	copy(source[1:], source)

	source[0] = item

	return source
}

//////
// Built-in options.
//////

// WithError allows to specify an error which will be wrapped by the custom
// error.
func WithError(err error) Option {
	return func(cE *CustomError) {
		cE.Err = err
	}
}

// WithMessage allows to specify the error message.
func WithMessage(msg string) Option {
	return func(cE *CustomError) {
		cE.Message = msg
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
//
//nolint:dupword
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
		if cE.Tags == nil {
			cE.Tags = &Set{treeset.NewWithStringComparator()}
		}

		for _, t := range tag {
			cE.Tags.Add(t)
		}
	}
}

// WithFields allows to set fields for the error.
func WithFields(fields map[string]interface{}) Option {
	return func(cE *CustomError) {
		if cE.Fields == nil {
			cE.Fields = &sync.Map{}
		}

		cE.Fields = mapToSyncMap(fields)
	}
}

// WithField allows to set a field for the error.
func WithField(key string, value any) Option {
	return func(cE *CustomError) {
		if cE.Fields == nil {
			cE.Fields = &sync.Map{}
		}

		cE.Fields.Store(key, value)
	}
}

// WithLanguage specifies the language for the error message.
// It requires `lang` to be a valid ISO 639-1 and ISO 3166-1 alpha-2 standard,
// and the `LanguageMessageMap` map to be set, otherwise it will be ignored
// returning the default message.
func WithLanguage(lang string) Option {
	return func(cE *CustomError) {
		l, err := NewLanguage(lang)
		if err != nil {
			panic(err)
		}

		if cE.LanguageMessageMap == nil {
			cE.LanguageMessageMap = &sync.Map{}
		}

		if msg, ok := cE.LanguageMessageMap.Load(l); ok {
			cE.SetMessage(msg.(string))
		}
	}
}

// WithTranslation sets translations for the error message.
func WithTranslation(lang, message string) Option {
	return func(cE *CustomError) {
		if cE.LanguageMessageMap == nil {
			cE.LanguageMessageMap = &sync.Map{}
		}

		l, err := NewLanguage(lang)
		if err != nil {
			panic(err)
		}

		cE.LanguageMessageMap.Store(l, message)
	}
}
