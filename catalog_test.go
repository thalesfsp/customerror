package customerror

import (
	"errors"
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
			if err := got.Add("INVALID_REQUEST_BODY", "invalid request body", WithTranslation("pt-BR", "corpo da solicitação inválido")); err != nil {
				t.Errorf("NewCatalog() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			// Example of adding a new language (es-ES) to an existing error
			// code.
			if err := got.Add("E1010", "invalid response", WithTranslation("es-ES", "resposta inválida")); err != nil {
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
