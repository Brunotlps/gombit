package contract

import (
	"reflect"
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

func TestFieldsFromErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		errs []error
		want map[string][]string
	}{
		{
			name: "nil and empty",
			errs: nil,
			want: nil,
		},
		{
			name: "body field strips prefix",
			errs: []error{
				&huma.ErrorDetail{Location: "body.name", Message: "expected length >= 1"},
			},
			want: map[string][]string{"name": {"expected length >= 1"}},
		},
		{
			name: "nested body path",
			errs: []error{
				&huma.ErrorDetail{Location: "body.items[0].tags", Message: "expected array"},
			},
			want: map[string][]string{"items[0].tags": {"expected array"}},
		},
		{
			name: "query path header keep prefix",
			errs: []error{
				&huma.ErrorDetail{Location: "query.limit", Message: "expected number"},
				&huma.ErrorDetail{Location: "path.widget-id", Message: "expected uuid"},
				&huma.ErrorDetail{Location: "header.x-token", Message: "required"},
			},
			want: map[string][]string{
				"query.limit":    {"expected number"},
				"path.widget-id": {"expected uuid"},
				"header.x-token": {"required"},
			},
		},
		{
			name: "multiple messages same field",
			errs: []error{
				&huma.ErrorDetail{Location: "body.email", Message: "required"},
				&huma.ErrorDetail{Location: "body.email", Message: "invalid format"},
			},
			want: map[string][]string{"email": {"required", "invalid format"}},
		},
		{
			name: "plain error falls back",
			errs: []error{errString("boom")},
			want: map[string][]string{"_error": {"boom"}},
		},
		{
			name: "required property inferred from message",
			errs: []error{
				&huma.ErrorDetail{Message: "expected required property name to be present"},
			},
			want: map[string][]string{"name": {"expected required property name to be present"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FieldsFromErrors(tt.errs...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("FieldsFromErrors() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

type errString string

func (e errString) Error() string { return string(e) }
