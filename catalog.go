package customerror

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

//////
// Consts, vars, and types.
//////

var (
	// ErrCatalogErrorNotFound is returned when a custom error isn't found in
	// the catalog.
	ErrCatalogErrorNotFound = NewNotFoundError("error", WithErrorCode("CE_ERR_CATALOG_ERR_NOT_FOUND"))

	// ErrCatalogInvalidName is returned when a catalog name is invalid.
	ErrCatalogInvalidName = NewInvalidError("name", WithErrorCode("CE_ERR_CATALOG_INVALID_NAME"))

	// ErrCatalogNilEntry is returned when the error being set resolves to nil
	// - for example, when an ignore option (`WithIgnoreFunc`/`WithIgnoreString`)
	// was triggered - and therefore cannot be stored in the catalog.
	ErrCatalogNilEntry = NewInvalidError("entry. It resolves to nil (an ignore option was triggered?) and cannot be stored in the catalog", WithErrorCode("CE_ERR_CATALOG_NIL_ENTRY"))

	// ErrErrorCodeInvalidCode is returned when an error code is invalid.
	ErrErrorCodeInvalidCode = NewInvalidError("error code. It requires typeOf, and subject", WithErrorCode("CE_ERR_INVALID_ERROR_CODE"))

	// ErrorCodeRegex validates error codes. A valid error code is a non-empty
	// string consisting solely of ASCII letters, digits, and underscores, for
	// example: "E1010", "INVALID_REQUEST", "ERR_A1_B2", "AbCd123". Spaces and
	// punctuation are rejected. The pattern is fully anchored so the entire
	// string must match (no partial matches).
	ErrorCodeRegex = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
)

type (
	// ErrorCode is the consistent way to express an error. Despite there's no
	// enforcement, it's recommended that to be meanginful, all upper cased and
	// separated by underscore, example: "INVALID_REQUEST".
	ErrorCode string

	// ErrorCodeErrorMap is a map of error codes to custom errors.
	ErrorCodeErrorMap = *sync.Map

	// Catalog contains a set of errors (customerrors).
	Catalog struct {
		// CustomErrors are the errors in the catalog.
		ErrorCodeErrorMap ErrorCodeErrorMap `json:"custom_errors"`

		// Name of the catalog, usually, the name of the application.
		Name string `json:"name" validate:"required,gte=3"`
	}
)

//////
// Methods.
//////

// String implements the Stringer interface.
func (e ErrorCode) String() string {
	return string(e)
}

// Validate if error code follows the pattern.
func (e ErrorCode) Validate() error {
	if !ErrorCodeRegex.MatchString(string(e)) {
		return ErrErrorCodeInvalidCode
	}

	return nil
}

// Set a custom error to the catalog. Use options to set default and common
// values such as fields, tags, etc. If the resulting error is nil (an ignore
// option was triggered), nothing is stored and `ErrCatalogNilEntry` is
// returned - a nil entry would otherwise panic on `Get`.
func (c *Catalog) Set(errorCode string, defaultMessage string, opts ...Option) (string, error) {
	eC, err := NewErrorCode(errorCode)
	if err != nil {
		return "", err
	}

	cE := Factory(defaultMessage, opts...)
	if cE == nil {
		return "", ErrCatalogNilEntry
	}

	c.ErrorCodeErrorMap.Store(eC, cE)

	return eC.String(), nil
}

// MustSet a custom error to the catalog. Use options to set default and common
// values such as fields, tags, etc. If an error occurs, panics.
func (c *Catalog) MustSet(errorCode string, defaultMessage string, opts ...Option) *Catalog {
	if _, err := c.Set(errorCode, defaultMessage, opts...); err != nil {
		panic(err)
	}

	return c
}

// Get returns a custom error from the catalog, if not found, returns an error.
//
// The returned `*CustomError` is a copy of the catalog entry, so callers can
// safely mutate it (or apply `opts`) without corrupting the shared catalog.
// Any `opts` provided are applied to the returned copy.
func (c *Catalog) Get(errorCode string, opts ...Option) (*CustomError, error) {
	errCode, err := NewErrorCode(errorCode)
	if err != nil {
		return nil, err
	}

	customErr, ok := c.ErrorCodeErrorMap.Load(errCode)
	if !ok {
		return nil, fmt.Errorf("%w. Code: %s", ErrCatalogErrorNotFound, errCode)
	}

	// Defensive: a non-CustomError or nil entry (e.g. stored directly into the
	// map, bypassing `Set`) is treated as not found instead of panicking.
	storedCE, ok := customErr.(*CustomError)
	if !ok || storedCE == nil {
		return nil, fmt.Errorf("%w. Code: %s", ErrCatalogErrorNotFound, errCode)
	}

	// Return a copy so the caller cannot mutate the catalog's stored entry.
	cE := Copy(storedCE, &CustomError{})

	// Apply the caller-provided options to the copy.
	for _, opt := range opts {
		opt(cE)
	}

	return cE, nil
}

// MustGet returns a custom error from the catalog, if not found, panics.
func (c *Catalog) MustGet(errorCode string, opts ...Option) *CustomError {
	customErr, err := c.Get(errorCode, opts...)
	if err != nil {
		panic(err)
	}

	return customErr
}

//////
// Factory.
//////

// NewErrorCode creates a new ErrorCode. It will be validated and stored upper
// cased.
func NewErrorCode(name string) (ErrorCode, error) {
	eC := ErrorCode(strings.ToUpper(name))

	if err := eC.Validate(); err != nil {
		return "", err
	}

	return eC, nil
}

// NewCatalog creates a new Catalog.
func NewCatalog(name string) (*Catalog, error) {
	c := &Catalog{
		ErrorCodeErrorMap: &sync.Map{},
		Name:              name,
	}

	if err := validate.Struct(c); err != nil {
		return nil, ErrCatalogInvalidName
	}

	return c, nil
}

// MustNewCatalog creates a new Catalog. If an error occurs, panics.
func MustNewCatalog(name string) *Catalog {
	c, err := NewCatalog(name)
	if err != nil {
		panic(err)
	}

	return c
}
