// Template provides a flexible way to define error messages in different
// languages.
//
// SEE: `i18n.md` file for more information.

package customerror

import (
	"fmt"
	"strings"
	"sync"
)

// These are the built-in languages supported by the error handling system, and
// their respective default error message.
const (
	FailedTo ErrorType = "failed to" // Indicates a failure to perform an action
	Invalid  ErrorType = "invalid"   // Denotes an invalid input or state
	Missing  ErrorType = "missing"   // Signifies a missing required element
	NotFound ErrorType = "not found" // Indicates that a requested resource was not found
	Required ErrorType = "required"  // Denotes a required element that was not provided
)

// Singleton variables and error instances.
var (
	once                      sync.Once
	singletonLanguageErrorMap LanguageErrorMap

	// ErrTemplateNotFound is returned when a requested error template is not
	// found in the map.
	ErrTemplateNotFound = NewNotFoundError(fmt.Sprintf(
		"%s. %s. Built-in languages: %s.",
		"template",
		"Please set one using `SetErrorPrefixMap`",
		strings.Join(BuiltInLanguages, ", "),
	), WithErrorCode("CE_ERR_TEMPLATE_NOT_FOUND"))

	// ErrLanguageNotFound is returned when a requested language is not supported.
	ErrLanguageNotFound = NewNotFoundError("language. Please set one using `SetErrorPrefixMap`", WithErrorCode("CE_ERR_LANGUAGE_NOT_FOUND"))
)

// Type definitions for the error handling system.
type (
	// ErrorType represents the category or nature of an error.
	ErrorType string

	// LanguageErrorMap is a thread-safe map storing error templates for
	// different languages.
	LanguageErrorMap = *sync.Map

	// ErrorPrefixMap is a thread-safe map storing error prefix templates for
	// different error types.
	ErrorPrefixMap = *sync.Map
)

// String implements the Stringer interface for ErrorType.
func (e ErrorType) String() string {
	return string(e)
}

// GetLanguageErrorMap returns the singleton instance of LanguageErrorMap.
// It initializes the map with built-in languages and their respective error
// templates.
//
//nolint:gosmopolitan,misspell
func GetLanguageErrorMap() LanguageErrorMap {
	once.Do(func() {
		languageErrorTypeMap := &sync.Map{}

		// Initialize error templates for English
		enErrorTypePrefixTemplateMap := &sync.Map{}
		enErrorTypePrefixTemplateMap.Store(FailedTo, "failed to %s")
		enErrorTypePrefixTemplateMap.Store(Invalid, "invalid %s")
		enErrorTypePrefixTemplateMap.Store(Missing, "missing %s")
		enErrorTypePrefixTemplateMap.Store(Required, "%s required")
		enErrorTypePrefixTemplateMap.Store(NotFound, "%s not found")

		enLanguage, err := NewLanguage(English.String())
		if err != nil {
			panic(err)
		}

		languageErrorTypeMap.Store(enLanguage, enErrorTypePrefixTemplateMap)

		// Initialize error templates for Chinese
		chErrorTypePrefixTemplateMap := &sync.Map{}
		chErrorTypePrefixTemplateMap.Store(FailedTo, "无法 %s")
		chErrorTypePrefixTemplateMap.Store(Invalid, "无效的 %s")
		chErrorTypePrefixTemplateMap.Store(Missing, "缺少 %s")
		chErrorTypePrefixTemplateMap.Store(Required, "需要 %s")
		chErrorTypePrefixTemplateMap.Store(NotFound, "%s 未找到")

		chLanguage, err := NewLanguage(Chinese.String())
		if err != nil {
			panic(err)
		}

		languageErrorTypeMap.Store(chLanguage, chErrorTypePrefixTemplateMap)

		// Initialize error templates for Spanish
		esErrorTypePrefixTemplateMap := &sync.Map{}
		esErrorTypePrefixTemplateMap.Store(FailedTo, "error al %s")
		esErrorTypePrefixTemplateMap.Store(Invalid, "%s inválido")
		esErrorTypePrefixTemplateMap.Store(Missing, "falta %s")
		esErrorTypePrefixTemplateMap.Store(Required, "%s requerido")
		esErrorTypePrefixTemplateMap.Store(NotFound, "%s no encontrado")

		esLanguage, err := NewLanguage(Spanish.String())
		if err != nil {
			panic(err)
		}

		languageErrorTypeMap.Store(esLanguage, esErrorTypePrefixTemplateMap)

		// Initialize error templates for French
		frErrorTypePrefixTemplateMap := &sync.Map{}
		frErrorTypePrefixTemplateMap.Store(FailedTo, "échec de %s")
		frErrorTypePrefixTemplateMap.Store(Invalid, "%s invalide")
		frErrorTypePrefixTemplateMap.Store(Missing, "%s manquant")
		frErrorTypePrefixTemplateMap.Store(Required, "%s requis")
		frErrorTypePrefixTemplateMap.Store(NotFound, "%s introuvable")

		frLanguage, err := NewLanguage(French.String())
		if err != nil {
			panic(err)
		}

		languageErrorTypeMap.Store(frLanguage, frErrorTypePrefixTemplateMap)

		// Initialize error templates for German
		deErrorTypePrefixTemplateMap := &sync.Map{}
		deErrorTypePrefixTemplateMap.Store(FailedTo, "fehlgeschlagen bei %s")
		deErrorTypePrefixTemplateMap.Store(Invalid, "ungültig %s")
		deErrorTypePrefixTemplateMap.Store(Missing, "fehlend %s")
		deErrorTypePrefixTemplateMap.Store(Required, "%s erforderlich")
		deErrorTypePrefixTemplateMap.Store(NotFound, "%s nicht gefunden")

		deLanguage, err := NewLanguage(German.String())
		if err != nil {
			panic(err)
		}

		languageErrorTypeMap.Store(deLanguage, deErrorTypePrefixTemplateMap)

		// Initialize error templates for Italian
		itErrorTypePrefixTemplateMap := &sync.Map{}
		itErrorTypePrefixTemplateMap.Store(FailedTo, "impossibile %s")
		itErrorTypePrefixTemplateMap.Store(Invalid, "%s non valido")
		itErrorTypePrefixTemplateMap.Store(Missing, "mancante %s")
		itErrorTypePrefixTemplateMap.Store(Required, "%s richiesto")
		itErrorTypePrefixTemplateMap.Store(NotFound, "%s non trovato")

		itLanguage, err := NewLanguage(Italian.String())
		if err != nil {
			panic(err)
		}

		languageErrorTypeMap.Store(itLanguage, itErrorTypePrefixTemplateMap)

		// Initialize error templates for Brazilian Portuguese
		ptBrErrorTypePrefixTemplateMap := &sync.Map{}
		ptBrErrorTypePrefixTemplateMap.Store(FailedTo, "falhou %s")
		ptBrErrorTypePrefixTemplateMap.Store(Invalid, "%s é inválido")
		ptBrErrorTypePrefixTemplateMap.Store(Missing, "faltando %s")
		ptBrErrorTypePrefixTemplateMap.Store(Required, "%s necessário")
		ptBrErrorTypePrefixTemplateMap.Store(NotFound, "%s não encontrado")

		ptLanguage, err := NewLanguage(Portuguese.String())
		if err != nil {
			panic(err)
		}

		languageErrorTypeMap.Store(ptLanguage, ptBrErrorTypePrefixTemplateMap)

		singletonLanguageErrorMap = languageErrorTypeMap
	})

	return singletonLanguageErrorMap
}

// GetLanguageErrorTypeMap retrieves the ErrorPrefixMap for a specific language.
// It returns an error if the language is not supported.
func GetLanguageErrorTypeMap(language string) (LanguageErrorMap, error) {
	l, err := NewLanguage(language)
	if err != nil {
		return nil, err
	}

	errorTypeMap, ok := GetLanguageErrorMap().Load(l)
	if !ok {
		return nil, ErrLanguageNotFound
	}

	return errorTypeMap.(LanguageErrorMap), nil
}

// GetTemplate retrieves the error template for a specific language and error type.
// It returns an error if either the language or the template is not found.
func GetTemplate(language, errorType string) (string, error) {
	languageErrorTypeMap, err := GetLanguageErrorTypeMap(language)
	if err != nil {
		return "", err
	}

	template, ok := languageErrorTypeMap.Load(ErrorType(errorType))
	if !ok {
		return "", ErrTemplateNotFound
	}

	return template.(string), nil
}

// AddNewLanguage registers (or updates) a language in the error handling
// system, setting the built-in error templates for each error type. If the
// language already exists, its templates are overwritten.
func AddNewLanguage(
	language string,
	errorTypePrefixTemplateMap ErrorPrefixMap,
) error {
	l, err := NewLanguage(language)
	if err != nil {
		return err
	}

	GetLanguageErrorMap().Store(l, errorTypePrefixTemplateMap)

	return nil
}

// MustAddNewLanguage is a wrapper around AddNewLanguage that panics if an error
// occurs.
func MustAddNewLanguage(
	language string,
	errorTypePrefixTemplateMap ErrorPrefixMap,
) {
	if err := AddNewLanguage(language, errorTypePrefixTemplateMap); err != nil {
		panic(err)
	}
}

// NewErrorPrefixMap creates a new ErrorPrefixMap with the provided templates for each error type.
// This function is useful when setting up custom error templates for a new language.
func NewErrorPrefixMap(
	failedToTemplate,
	invalidTemplate,
	missingTemplate,
	requiredTemplate,
	notFoundTemplate string,
) ErrorPrefixMap {
	errorTypePrefixTemplateMap := &sync.Map{}

	errorTypePrefixTemplateMap.Store(FailedTo, failedToTemplate)
	errorTypePrefixTemplateMap.Store(Invalid, invalidTemplate)
	errorTypePrefixTemplateMap.Store(Missing, missingTemplate)
	errorTypePrefixTemplateMap.Store(Required, requiredTemplate)
	errorTypePrefixTemplateMap.Store(NotFound, notFoundTemplate)

	return errorTypePrefixTemplateMap
}
