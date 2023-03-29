package customerror

import (
	"errors"
	"net/http"
	"testing"
)

func TestNewCatalog(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    *Catalog
		wantErr bool
	}{
		{
			name: "Should work",
			args: args{
				name: "myapp",
			},
			want:    &Catalog{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCatalog(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCatalog() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Example of adding a new language (pt-BR) to an existing error
			// code.
			if err := got.Set("INVALID_REQUEST_BODY", "invalid request body", WithTranslation("pt-BR", "corpo da solicitação inválido")); err != nil {
				t.Errorf("NewCatalog() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			// Example of adding a new language (es-ES) to an existing error
			// code.
			if err := got.Set("E1010", "invalid response", WithTranslation("es-ES", "resposta inválida")); err != nil {
				t.Errorf("NewCatalog() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			cEInvalidRequestBodyErr, err := got.Get("INVALID_REQUEST_BODY")
			if err != nil {
				t.Errorf("NewCatalog() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			// Example of throwing the error in pt-BR, also wrapping another
			// error.
			cEInvalidRequestBody := cEInvalidRequestBodyErr.New(WithLanguage("pt-BR"), WithError(errors.New("some error")))

			if cEInvalidRequestBody.Error() != "corpo da solicitação inválido. Original Error: some error" {
				t.Errorf("NewCatalog() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			cEE1010Err, err := got.Get("E1010")
			if err != nil {
				t.Errorf("NewCatalog() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			cEE1010 := cEE1010Err.New(WithLanguage("es-ES"))

			if cEE1010.Error() != "resposta inválida" {
				t.Errorf("NewCatalog() error = %v, wantErr %v", err, "resposta inválida")

				return
			}
		})
	}
}

func TestNewCatalog_2(t *testing.T) {
	cE := Factory("insert id", WithTranslation("pt-BR", "id"))

	x1 := cE.NewFailedToError(WithLanguage("pt-BR"))

	if x1.Error() != "falhou id" {
		t.Fatalf("Got %s Expected %s", x1, "falhou id")
	}

	x2 := cE.NewInvalidError(WithLanguage("pt-BR"))

	if x2.Error() != "id é inválido" {
		t.Fatalf("Got %s Expected %s", x2, "id é inválido")
	}

	x3 := cE.NewMissingError(WithLanguage("pt-BR"))

	if x3.Error() != "faltando id" {
		t.Fatalf("Got %s Expected %s", x3, "faltando id")
	}

	x4 := cE.NewRequiredError(WithLanguage("pt-BR"))

	if x4.Error() != "id necessário" {
		t.Fatalf("Got %s Expected %s", x4, "id necessário")
	}

	x5 := cE.NewHTTPError(http.StatusNoContent, WithLanguage("pt-BR"))

	if x5.Error() != "no content" {
		t.Fatalf("Got %s Expected %s", x5, "no content")
	}
}
