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
//
// It returns a brand-new slice with `item` first, followed by `source`. A
// fresh slice is always allocated so the caller's backing array is never
// mutated (avoids append/copy aliasing bugs).
func prependOptions(source []Option, item Option) []Option {
	result := make([]Option, 0, len(source)+1)

	result = append(result, item)
	result = append(result, source...)

	return result
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

// WithErrorCode allows to specify an error code, such as "E1010".
func WithErrorCode(code string) Option {
	return func(cE *CustomError) {
		cE.Code = code
	}
}

// WithRetryable allows to specify if the error is retryable.
func WithRetryable(status bool) Option {
	return func(cE *CustomError) {
		cE.Retryable = status
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

// WithFields allows to set fields for the error. Existing fields are preserved;
// keys present in `fields` are added or overwritten (consistent with
// `WithField`).
func WithFields(fields map[string]interface{}) Option {
	return func(cE *CustomError) {
		if cE.Fields == nil {
			cE.Fields = &sync.Map{}
		}

		for k, v := range fields {
			cE.Fields.Store(k, v)
		}
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
// returning the default message. If a language is specified in the "en-US"
// format, and not found, it will try to find by the root "en".
func WithLanguage(lang string) Option {
	return func(cE *CustomError) {
		l, err := NewLanguage(lang)
		if err != nil {
			// Invalid language code: ignore it and keep the default message
			// (as documented) instead of panicking.
			return
		}

		if cE.LanguageMessageMap == nil {
			cE.LanguageMessageMap = &sync.Map{}
		}

		if msg, ok := cE.LanguageMessageMap.Load(l); ok {
			cE.language = l

			cE.SetMessage(msg.(string))
		}
	}
}

// WithTranslation sets the translation for the error message.
//
// NOTE: For supported built-in languages, the default word(s) used by
// the built-in error functions (example: `NewFailedToError`), are
// automatically translated and included in the message.
//
// SEE: `languages.go` file an up-to-date list of the supported built-in
// list of languages.
//
// For ANY other language there are two options:
// 1. Don't use the built-in functions but instead use the `New` function
// and write the message in full, for example: "invalid hard drive path".
// 2. Setup the language (see `ExampleNew_i18nSetupNewLang`).
//
// Reason: It's impossible for any package to cover all the possible
// languages, combinations, and their translations.
//
// SEE: `i18n.md` file for more information.
//
// NOTE: If `lang` is not a valid language code, the translation is ignored
// (not stored) instead of panicking.
func WithTranslation(lang, message string) Option {
	return func(cE *CustomError) {
		l, err := NewLanguage(lang)
		if err != nil {
			// Invalid language code: ignore it instead of panicking.
			return
		}

		if cE.LanguageMessageMap == nil {
			cE.LanguageMessageMap = &sync.Map{}
		}

		cE.LanguageMessageMap.Store(l, message)
	}
}
