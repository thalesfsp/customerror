package customerror

import (
	"fmt"
	"strings"
	"sync"
)

//////
// Consts, vars, and types.
//////

// Error types.
const (
	FailedTo ErrorType = "failed to"
	Invalid  ErrorType = "invalid"
	Missing  ErrorType = "missing"
	NotFound ErrorType = "not found"
	Required ErrorType = "required"
)

// Singleton.
var (
	once                      sync.Once
	singletonLanguageErrorMap LanguageErrorMap

	// ErrTemplateNotFound is returned when a template isn't found in the map.
	ErrTemplateNotFound = NewNotFoundError(fmt.Sprintf(
		"%s. %s. Built-in languages: %s.",
		"template",
		"Please set one using `SetErrorPrefixMap`",
		strings.Join(BuiltInLanguages, ", "),
	), WithCode("CE_ERR_TEMPLATE_NOT_FOUND"))

	// ErrLanguageNotFound is returned when a language isn't found in the map.
	ErrLanguageNotFound = NewNotFoundError("language. Please set one using `SetErrorPrefixMap`", WithCode("CE_ERR_LANGUAGE_NOT_FOUND"))
)

type (
	// ErrorType is the type of the error.
	ErrorType string

	// LanguageErrorMap is a map of language to types of errors.
	LanguageErrorMap = *sync.Map

	// ErrorPrefixMap is a map of types of errors to prefixes templates such as
	// "missing %s", "%s required", "%s invalid", etc.
	ErrorPrefixMap = *sync.Map
)

//////
// Methods.
//////

// String implements the Stringer interface.
func (e ErrorType) String() string {
	return string(e)
}

//////
// Exported functionalities.
//////

// GetLanguageErrorMap returns the language prefix template map.
func GetLanguageErrorMap() LanguageErrorMap {
	once.Do(func() {
		languageErrorTypeMap := &sync.Map{}

		//////
		// English.
		//////

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

		//////
		// Chinese.
		//////

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

		//////
		// Spanish.
		//////

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

		//////
		// French.
		//////

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

		//////
		// German.
		//////

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

		//////
		// Italian.
		//////

		itErrorTypePrefixTemplateMap := &sync.Map{}

		itErrorTypePrefixTemplateMap.Store(FailedTo, "impossible %s")
		itErrorTypePrefixTemplateMap.Store(Invalid, "%s non valido")
		itErrorTypePrefixTemplateMap.Store(Missing, "mancante %s")
		itErrorTypePrefixTemplateMap.Store(Required, "%s richiesto")
		itErrorTypePrefixTemplateMap.Store(NotFound, "%s non trovato")

		itLanguage, err := NewLanguage(Italian.String())
		if err != nil {
			panic(err)
		}

		languageErrorTypeMap.Store(itLanguage, itErrorTypePrefixTemplateMap)

		//////
		// Brazilian Portuguese.
		//////

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

// GetLanguageErrorTypeMap returns the language error type map.
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

// GetTemplate returns the template for the given language and error type.
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

// SetErrorPrefixMap sets the error type prefix template map for the
// given language.
func SetErrorPrefixMap(
	language string,
	errorTypePrefixTemplateMap ErrorPrefixMap,
) error {
	l, err := NewLanguage(language)
	if err != nil {
		return err
	}

	GetLanguageErrorMap().LoadOrStore(l, errorTypePrefixTemplateMap)

	return nil
}

// NewErrorPrefixMap returns a new error type prefix template map.
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
