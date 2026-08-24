package admin

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

type setterRow struct {
	Note    *string
	Name    string
	Payload json.RawMessage
	When    time.Time
	WhenPtr *time.Time
}

func TestConvertToNilMatchesDest(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		dest reflect.Type
		want func(reflect.Value) bool
	}{
		{
			name: "string",
			dest: reflect.TypeOf(""),
			want: func(v reflect.Value) bool { return v.Kind() == reflect.String && v.String() == "" },
		},
		{
			name: "pointer string",
			dest: reflect.TypeOf((*string)(nil)),
			want: func(v reflect.Value) bool { return v.Kind() == reflect.Pointer && v.IsNil() },
		},
		{
			name: "time.Time",
			dest: reflect.TypeOf(time.Time{}),
			want: func(v reflect.Value) bool {
				got, ok := v.Interface().(time.Time)
				return ok && got.IsZero()
			},
		},
		{
			name: "pointer time.Time",
			dest: reflect.TypeOf((*time.Time)(nil)),
			want: func(v reflect.Value) bool { return v.Kind() == reflect.Pointer && v.IsNil() },
		},
		{
			name: "json.RawMessage",
			dest: reflect.TypeOf(json.RawMessage{}),
			want: func(v reflect.Value) bool {
				got, ok := v.Interface().(json.RawMessage)
				return ok && got == nil
			},
		},
		{
			name: "[]byte",
			dest: reflect.TypeOf([]byte(nil)),
			want: func(v reflect.Value) bool {
				got, ok := v.Interface().([]byte)
				return ok && got == nil
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := convertTo(nil, tc.dest)
			if err != nil {
				t.Fatalf("convertTo(nil, %s) error = %v", tc.dest, err)
			}
			if !tc.want(got) {
				t.Fatalf("convertTo(nil, %s) = %#v, want dest zero/nil", tc.dest, got.Interface())
			}
		})
	}
}

func TestMakeSetterClearsPresentNil(t *testing.T) {
	t.Parallel()
	note := "hello"
	when := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	inst := &setterRow{
		Note:    &note,
		Name:    "keep",
		Payload: json.RawMessage(`{"a":1}`),
		When:    when,
		WhenPtr: &when,
	}
	rt := reflect.TypeOf(setterRow{})

	clear := func(name string, ft FieldType, raw any) {
		t.Helper()
		sf, ok := rt.FieldByName(name)
		if !ok {
			t.Fatalf("field %s missing", name)
		}
		set := makeSetter(sf.Index, ft, sf.Type)
		if err := set(inst, raw); err != nil {
			t.Fatalf("set %s: %v", name, err)
		}
	}

	clear("Note", TypeText, nil)
	clear("Payload", TypeJSON, nil)
	clear("When", TypeDateTime, nil)
	clear("WhenPtr", TypeDateTime, nil)

	if inst.Note != nil {
		t.Fatalf("Note = %#v, want nil pointer", inst.Note)
	}
	if inst.Name != "keep" {
		t.Fatalf("Name = %q, want %q (omitted key must stay unchanged)", inst.Name, "keep")
	}
	if inst.Payload != nil {
		t.Fatalf("Payload = %#v, want nil", inst.Payload)
	}
	if !inst.When.IsZero() {
		t.Fatalf("When = %s, want zero time", inst.When)
	}
	if inst.WhenPtr != nil {
		t.Fatalf("WhenPtr = %#v, want nil pointer", inst.WhenPtr)
	}

	clear("Name", TypeString, "")
	if inst.Name != "" {
		t.Fatalf("Name after empty string = %q, want empty", inst.Name)
	}
	clear("Name", TypeString, nil)
	if inst.Name != "" {
		t.Fatalf("Name after null = %q, want empty", inst.Name)
	}
}
