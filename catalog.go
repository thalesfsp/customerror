package customerror

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

//////
// Consts, vars, and types.
//////

var (
	// ErrCatalogErrorNotFound is returned when a custom error isn't found in
	// the catalog.
	ErrCatalogErrorNotFound = NewNotFoundError("error", WithCode("CE_ERR_CATALOG_ERR_NOT_FOUND"))

	// ErrCatalogInvalidName is returned when a catalog name is invalid.
	ErrCatalogInvalidName = NewInvalidError("name", WithCode("CE_ERR_CATALOG_INVALID_NAME"))

	// ErrErrorCodeInvalidCode is returned when an error code is invalid.
	ErrErrorCodeInvalidCode = NewInvalidError("error code. It requires typeOf, and subject", WithCode("CE_ERR_INVALID_ERROR_CODE"))

	// ErrorCodeRegex is a regular expression to validate error codes. It's
	// designed to match four distinct patterns:
	//
	// 1. An optional number and letter followed by an underscore, "ERR_", a
	// letter and a number, another underscore, another letter and a number, and
	// an optional underscore followed by a number and a letter:
	// Format: {optional_number_letter}_ERR_{number_letter}_{number_letter}_{optional_number_letter}
	// Example: 1A_ERR_A1_B2 or 1A_ERR_A1_B2_3C
	//
	// 2. "ERR_" followed by a letter and a number, another underscore, another
	// letter and a number, and an optional underscore followed by a number and
	// a letter:
	// Format: ERR_{number_letter}_{number_letter}_{optional_number_letter}
	// Example: ERR_A1_B2 or ERR_A1_B2_3C
	//
	// 3. "E" followed by 1 to 8 digits:
	// Format: E{1 to 8 digits}
	// Example: E12345678
	//
	// 4. At least one letter or number (any combination of uppercase and
	// lowercase letters and digits):
	// Format: {letters or digits, at least one character}
	// Example: AbCd123.
	ErrorCodeRegex = regexp.MustCompile(`^(\d?[A-Za-z]_)?ERR_[A-Za-z]\d_[A-Za-z]\d(_\d[A-Za-z])?$|^ERR_[A-Za-z]\d_[A-Za-z]\d(_\d[A-Za-z])?$|^E\d{1,8}$|[A-Za-z\d]+`)
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
		Name string `json:"name" validate:"required,gt=3"`
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
// values such as fields, tags, etc.
func (c *Catalog) Set(errorCode string, defaultMessage string, opts ...Option) (string, error) {
	eC, err := NewErrorCode(errorCode)
	if err != nil {
		return "", err
	}

	c.ErrorCodeErrorMap.Store(eC, Factory(defaultMessage, opts...))

	return eC.String(), nil
}

// MustSet a custom error to the catalog. Use options to set default and common
// values such as fields, tags, etc. If an error occurs, panics.
func (c *Catalog) MustSet(errorCode string, defaultMessage string, opts ...Option) string {
	ec, err := c.Set(errorCode, defaultMessage, opts...)
	if err != nil {
		panic(err)
	}

	return ec
}

// Get returns a custom error from the catalog, if not found, returns an error.
func (c *Catalog) Get(errorCode string, opts ...Option) (*CustomError, error) {
	errCode, err := NewErrorCode(errorCode)
	if err != nil {
		return nil, err
	}

	customErr, ok := c.ErrorCodeErrorMap.Load(errCode)
	if ok {
		return customErr.(*CustomError), nil
	}

	return nil, fmt.Errorf("%w. Code: %s", ErrCatalogErrorNotFound, errCode)
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

	if err := validator.New().Struct(c); err != nil {
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
