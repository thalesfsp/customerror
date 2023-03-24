package customerror

import (
	"regexp"
	"sync"
)

//////
// Consts, vars, and types.
//////

var (
	// ErrInvalidLangCode is returned when a language code is invalid.
	ErrInvalidLangCode = NewInvalidError("it must be a string, two-letter lowercase ISO 639-1 code OR two-letter lowercase ISO 639-1 code followed by an optional hyphen AND a two-letter uppercase ISO 3166-1 alpha-2 country code", WithCode("CE_ERR_INVALID_LANG_CODE"))

	// ErrInvalidLangErrorMessage is returned when an error message is invalid.
	ErrInvalidLangErrorMessage = NewInvalidError("it must be a string, at least 3 characters long", WithCode("CE_ERR_INVALID_LANG_ERROR_MESSAGE"))

	// ErrInvalidLanguageMessageMap is returned when a LanguageMessageMap is
	// invalid.
	ErrInvalidLanguageMessageMap = NewInvalidError("it must be a non-nil map of language codes to error messages", WithCode("CE_ERR_INVALID_LANGUAGE_MESSAGE_MAP"))

	// LanguageRegex is a regular expression to validate language codes based on
	// ISO 639-1 and ISO 3166-1 alpha-2.
	LanguageRegex = regexp.MustCompile("^[a-z]{2}(-[A-Z]{2})?$|default")
)

type (
	// Language is a language code.
	Language string

	// LanguageMessageMap is a map of language codes to error messages.
	LanguageMessageMap = *sync.Map
)

//////
// Methods.
//////

// String implements the Stringer interface.
func (l Language) String() string {
	return string(l)
}

// Validate if lang follows the ISO 639-1 and ISO 3166-1 alpha-2 standard.
func (l Language) Validate() error {
	if !LanguageRegex.MatchString(l.String()) {
		return ErrInvalidLangCode
	}

	return nil
}

//////
// Factory.
//////

// NewLanguage creates a new Lang.
func NewLanguage(lang string) (Language, error) {
	l := Language(lang)

	if err := l.Validate(); err != nil {
		return "", err
	}

	return l, nil
}
