package customerror

import (
	"testing"
)

func TestGetTemplate(t *testing.T) {
	type args struct {
		language  string
		errorType string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Should work",
			args: args{
				language:  "en",
				errorType: string(FailedTo),
			},
			want:    "failed to %s",
			wantErr: false,
		},
		{
			name: "Should work",
			args: args{
				language:  "es",
				errorType: string(FailedTo),
			},
			want:    "error al %s",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTemplate(tt.args.language, tt.args.errorType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetErrorTypePrefixTemplateMap(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Should work",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetErrorPrefixMap("bl-BA", NewErrorPrefixMap(
				"blablabla %s",
				"blebleble %s",
				"bliblibli %s",
				"blobloblo %s",
				"blublublu %s",
			)); (err != nil) != tt.wantErr {
				t.Errorf("SetErrorTypePrefixTemplateMap() error = %v, wantErr %v", err, tt.wantErr)
			}

			tFailedTo, err := GetTemplate("bl-BA", string(FailedTo))
			if err != nil {
				t.Error(err)

				return
			}

			if tFailedTo != "blablabla %s" {
				t.Error("Expected", "blablabla %s")

				return
			}

			tInvalid, err := GetTemplate("bl-BA", string(Invalid))
			if err != nil {
				t.Error(err)

				return
			}

			if tInvalid != "blebleble %s" {
				t.Error("Expected", "blebleble %s")

				return
			}

			tMissing, err := GetTemplate("bl-BA", string(Missing))
			if err != nil {
				t.Error(err)

				return
			}

			if tMissing != "bliblibli %s" {
				t.Error("Expected", "bliblibli %s")

				return
			}

			tRequired, err := GetTemplate("bl-BA", string(Required))
			if err != nil {
				t.Error(err)

				return
			}

			if tRequired != "blobloblo %s" {
				t.Error("Expected", "blobloblo %s")

				return
			}

			tNotFound, err := GetTemplate("bl-BA", string(NotFound))
			if err != nil {
				t.Error(err)

				return
			}

			if tNotFound != "blublublu %s" {
				t.Error("Expected", "blublublu %s")

				return
			}
		})
	}
}
