package customerror

import (
	"regexp"
	"strings"
	"sync"
)

//////
// Consts, vars, and types.
//////

const (
	Chinese    Language = "ch"
	English    Language = "en"
	French     Language = "fr"
	German     Language = "de"
	Italian    Language = "it"
	Portuguese Language = "pt"
	Spanish    Language = "es"
)

var (
	// ErrInvalidLanguageCode is returned when a language code is invalid.
	ErrInvalidLanguageCode = NewInvalidError("it must be a string, two-letter lowercase ISO 639-1 code OR two-letter lowercase ISO 639-1 code followed by an optional hyphen AND a two-letter uppercase ISO 3166-1 alpha-2 country code", WithCode("CE_ERR_INVALID_LANG_CODE"))

	// ErrInvalidLanguageErrorMessage is returned when an error message is invalid.
	ErrInvalidLanguageErrorMessage = NewInvalidError("it must be a string, at least 3 characters long", WithCode("CE_ERR_INVALID_LANG_ERROR_MESSAGE"))

	// ErrInvalidLanguageMessageMap is returned when a LanguageMessageMap is
	// invalid.
	ErrInvalidLanguageMessageMap = NewInvalidError("it must be a non-nil map of language codes to error messages", WithCode("CE_ERR_INVALID_LANGUAGE_MESSAGE_MAP"))

	// BuiltInLanguages is a list of built-in prefixes languages.
	BuiltInLanguages = []string{
		Chinese.String(),
		English.String(),
		French.String(),
		German.String(),
		Italian.String(),
		Portuguese.String(),
		Spanish.String(),
	}

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
		return ErrInvalidLanguageCode
	}

	return nil
}

// GetRoot returns the root language code. Given "en-US", it returns "en".
func (l Language) GetRoot() string {
	matches := LanguageRegex.FindStringSubmatch(l.String())

	if len(matches) > 1 {
		return strings.ReplaceAll(matches[0], matches[1], "")
	}

	return ""
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
