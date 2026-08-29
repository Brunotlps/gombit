package resourcegen

import (
	"strings"
	"testing"
)

func TestParseFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		spec    string
		want    Field
		wantErr string
	}{
		{
			name: "required string",
			spec: "title:string:required",
			want: Field{JSONName: "title", GoName: "Title", Type: FieldString, GoType: "string", Required: true},
		},
		{
			name: "int with unique and index",
			spec: "sku:string:required,unique,index",
			want: Field{JSONName: "sku", GoName: "Sku", Type: FieldString, GoType: "string", Required: true, Unique: true, Index: true},
		},
		{
			name: "uint index",
			spec: "customer_id:uint:required,index",
			want: Field{JSONName: "customer_id", GoName: "CustomerID", Type: FieldUint, GoType: "uint", Required: true, Index: true},
		},
		{
			name: "bool",
			spec: "paid:bool",
			want: Field{JSONName: "paid", GoName: "Paid", Type: FieldBool, GoType: "bool"},
		},
		{
			name: "text",
			spec: "body:text",
			want: Field{JSONName: "body", GoName: "Body", Type: FieldText, GoType: "string"},
		},
		{
			name: "int64",
			spec: "price:int64",
			want: Field{JSONName: "price", GoName: "Price", Type: FieldInt64, GoType: "int64"},
		},
		{
			name: "decimal required (default precision)",
			spec: "amount:decimal:required",
			want: Field{JSONName: "amount", GoName: "Amount", Type: FieldDecimal, GoType: "types.Decimal", Required: true},
		},
		{
			name: "decimal with precision",
			spec: "price:decimal(10,2)",
			want: Field{JSONName: "price", GoName: "Price", Type: FieldDecimal, GoType: "types.Decimal"},
		},
		{
			name: "time",
			spec: "starts_at:time:required",
			want: Field{JSONName: "starts_at", GoName: "StartsAt", Type: FieldTime, GoType: "time.Time", Required: true},
		},
		{
			name: "enum",
			spec: "status:enum(requested,confirmed,active)",
			want: Field{JSONName: "status", GoName: "Status", Type: FieldEnum, GoType: "string"},
		},
		{
			name:    "unknown type",
			spec:    "amount:blob:required",
			wantErr: "unknown type",
		},
		{
			name:    "enum without values",
			spec:    "status:enum",
			wantErr: "needs values",
		},
		{
			name:    "decimal bad precision",
			spec:    "price:decimal(2,5)",
			wantErr: "invalid decimal",
		},
		{
			name:    "reserved gorm field",
			spec:    "id:uint",
			wantErr: "gorm.Model",
		},
		{
			name:    "unknown modifier",
			spec:    "name:string:blarg",
			wantErr: "unknown modifier",
		},
		{
			name:    "missing type",
			spec:    "name",
			wantErr: "name:type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fields, err := parseFields([]string{tt.spec})
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("parseFields() error = nil, want error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseFields() error = %q, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseFields() error = %v", err)
			}
			if len(fields) != 1 {
				t.Fatalf("len(fields) = %d, want 1", len(fields))
			}
			got := fields[0]
			if got.JSONName != tt.want.JSONName || got.GoName != tt.want.GoName || got.Type != tt.want.Type ||
				got.GoType != tt.want.GoType || got.Required != tt.want.Required ||
				got.Unique != tt.want.Unique || got.Index != tt.want.Index {
				t.Fatalf("field = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseFieldsDuplicate(t *testing.T) {
	t.Parallel()
	_, err := parseFields([]string{"title:string", "title:int"})
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("error = %v, want duplicate", err)
	}
}

func TestParseEnumFieldDetails(t *testing.T) {
	t.Parallel()
	// Enum values are case-sensitive and preserve declared order.
	fields, err := parseFields([]string{"status:enum(Draft,Published,Archived)"})
	if err != nil {
		t.Fatalf("parseFields() error = %v", err)
	}
	f := fields[0]
	want := []string{"Draft", "Published", "Archived"}
	if strings.Join(f.EnumValues, ",") != strings.Join(want, ",") {
		t.Fatalf("EnumValues = %v, want %v", f.EnumValues, want)
	}
	if got := f.humaTags(); !strings.Contains(got, `enum:"Draft,Published,Archived"`) {
		t.Fatalf("humaTags = %q, want enum tag", got)
	}
	if got := f.gormTag(); !strings.Contains(got, "size:") {
		t.Fatalf("gormTag = %q, want a sized varchar column", got)
	}
}

func TestParseDecimalPrecision(t *testing.T) {
	t.Parallel()
	fields, err := parseFields([]string{"amount:decimal", "price:decimal(10,2):required"})
	if err != nil {
		t.Fatalf("parseFields() error = %v", err)
	}
	if fields[0].Precision != 19 || fields[0].Scale != 4 {
		t.Fatalf("default decimal = (%d,%d), want (19,4)", fields[0].Precision, fields[0].Scale)
	}
	if got := fields[0].gormTag(); !strings.Contains(got, "type:decimal(19,4)") {
		t.Fatalf("gormTag = %q, want type:decimal(19,4)", got)
	}
	if fields[1].Precision != 10 || fields[1].Scale != 2 {
		t.Fatalf("decimal(10,2) = (%d,%d)", fields[1].Precision, fields[1].Scale)
	}
	if got := fields[1].gormTag(); !strings.Contains(got, "type:decimal(10,2)") || !strings.Contains(got, "not null") {
		t.Fatalf("gormTag = %q, want type:decimal(10,2);not null", got)
	}
}

func TestParseDuplicateEnumValueRejected(t *testing.T) {
	t.Parallel()
	_, err := parseFields([]string{"status:enum(a,b,a)"})
	if err == nil || !strings.Contains(err.Error(), "duplicate enum value") {
		t.Fatalf("error = %v, want duplicate enum value", err)
	}
}

func TestParseResourceName(t *testing.T) {
	t.Parallel()
	got, err := parseResourceName("Book")
	if err != nil {
		t.Fatalf("parseResourceName() error = %v", err)
	}
	if got.Package != "book" || got.TypeName != "Book" || got.HTTPPath != "/books" {
		t.Fatalf("got %+v", got)
	}

	widget, err := parseResourceName("Widget")
	if err != nil {
		t.Fatalf("Widget: %v", err)
	}
	if widget.HTTPPath != "/widgets" || widget.Package != "widget" {
		t.Fatalf("widget = %+v", widget)
	}

	bus, err := parseResourceName("Bus")
	if err != nil {
		t.Fatalf("Bus: %v", err)
	}
	buse, err := parseResourceName("Buse")
	if err != nil {
		t.Fatalf("Buse: %v", err)
	}
	if bus.HTTPPath != "/buses" || buse.HTTPPath != "/buses" {
		t.Fatalf("Bus HTTPPath = %q, Buse HTTPPath = %q, want both /buses", bus.HTTPPath, buse.HTTPPath)
	}
	if bus.Package == buse.Package {
		t.Fatalf("Bus and Buse must stay distinct packages, got %q", bus.Package)
	}

	box, err := parseResourceName("Box")
	if err != nil {
		t.Fatalf("Box: %v", err)
	}
	boxe, err := parseResourceName("Boxe")
	if err != nil {
		t.Fatalf("Boxe: %v", err)
	}
	if box.HTTPPath != "/boxes" || boxe.HTTPPath != "/boxes" {
		t.Fatalf("Box HTTPPath = %q, Boxe HTTPPath = %q, want both /boxes", box.HTTPPath, boxe.HTTPPath)
	}

	_, err = parseResourceName("platform")
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("platform error = %v, want reserved", err)
	}

	_, err = parseResourceName("web")
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("web error = %v, want reserved", err)
	}

	_, err = parseResourceName("Product")
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("Product error = %v, want reserved package product", err)
	}
	_, err = parseResourceName("product")
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("product error = %v, want reserved package product", err)
	}
}
