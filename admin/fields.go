package admin

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// FieldsFrom derives a default []Field from model at registration time.
// It may use reflect on this one type. Do not call it from request handlers.
//
// Name comes from the json tag when present, otherwise the GORM column /
// snake_case of the exported field name. Type is inferred from the Go type.
// Primary keys are readonly. GORM created_at / updated_at are included when
// present on the struct.
func FieldsFrom(model any) ([]Field, error) {
	sch, err := parseSchema(model)
	if err != nil {
		return nil, err
	}
	fields := make([]Field, 0, len(sch.Fields))
	for _, sf := range sch.Fields {
		if sf == nil || !sf.StructField.IsExported() {
			continue
		}
		if sf.DBName == "" {
			continue
		}
		if sf.StructField.Type == reflect.TypeOf(gorm.DeletedAt{}) {
			continue
		}
		name := jsonFieldName(sf)
		if name == "" || name == "-" {
			continue
		}
		ft := inferFieldType(sf)
		readOnly := sf.PrimaryKey || sf.AutoIncrement || sf.AutoCreateTime > 0 || sf.AutoUpdateTime > 0
		required := sf.NotNull && !readOnly && !sf.HasDefaultValue && sf.FieldType.Kind() != reflect.Pointer
		fields = append(fields, Field{
			Name:     name,
			Type:     ft,
			Required: required,
			ReadOnly: readOnly,
			Column:   sf.DBName,
		})
	}
	return fields, nil
}

func parseSchema(model any) (*schema.Schema, error) {
	if model == nil {
		return nil, fmt.Errorf("admin: nil model")
	}
	rv := reflect.ValueOf(model)
	if rv.Kind() != reflect.Pointer {
		ptr := reflect.New(rv.Type())
		ptr.Elem().Set(rv)
		model = ptr.Interface()
	} else if rv.IsNil() {
		model = reflect.New(rv.Type().Elem()).Interface()
	}
	sch, err := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		return nil, fmt.Errorf("admin: parse schema: %w", err)
	}
	return sch, nil
}

func jsonFieldName(sf *schema.Field) string {
	tag := sf.StructField.Tag.Get("json")
	if tag != "" {
		name, _, _ := strings.Cut(tag, ",")
		name = strings.TrimSpace(name)
		if name != "" {
			return name
		}
	}
	if sf.DBName != "" {
		return sf.DBName
	}
	return toSnake(sf.Name)
}

func inferFieldType(sf *schema.Field) FieldType {
	t := sf.FieldType
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t {
	case reflect.TypeOf(time.Time{}):
		return TypeDateTime
	case reflect.TypeOf(json.RawMessage{}):
		return TypeJSON
	}
	switch t.Kind() {
	case reflect.String:
		if strings.Contains(strings.ToLower(string(sf.DataType)), "text") {
			return TypeText
		}
		return TypeString
	case reflect.Bool:
		return TypeBoolean
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return TypeInteger
	case reflect.Float32, reflect.Float64:
		return TypeFloat
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
		return TypeJSON
	default:
		return TypeString
	}
}

func derivePK(sch *schema.Schema) (name, column string, ft FieldType, err error) {
	for _, sf := range sch.PrimaryFields {
		if sf == nil {
			continue
		}
		return jsonFieldName(sf), sf.DBName, inferFieldType(sf), nil
	}
	for _, sf := range sch.Fields {
		if sf != nil && sf.PrimaryKey {
			return jsonFieldName(sf), sf.DBName, inferFieldType(sf), nil
		}
	}
	return "", "", "", fmt.Errorf("admin: model has no primary key")
}

func matchSchemaField(sch *schema.Schema, f Field) *schema.Field {
	for _, sf := range sch.Fields {
		if sf == nil {
			continue
		}
		if f.Column != "" && sf.DBName == f.Column {
			return sf
		}
		if jsonFieldName(sf) == f.Name || sf.DBName == f.Name || sf.Name == f.Name {
			return sf
		}
	}
	return nil
}

func implicitTimestampColumns(sch *schema.Schema) map[string]implicitColumn {
	out := map[string]implicitColumn{}
	for _, sf := range sch.Fields {
		if sf == nil {
			continue
		}
		var name string
		switch {
		case sf.AutoCreateTime > 0 || sf.DBName == ImplicitCreatedAt:
			name = ImplicitCreatedAt
		case sf.AutoUpdateTime > 0 || sf.DBName == ImplicitUpdatedAt:
			name = ImplicitUpdatedAt
		default:
			continue
		}
		if sf.DBName == "" {
			continue
		}
		out[name] = implicitColumn{
			column: sf.DBName,
			get:    makeGetter(sf.StructField.Index, TypeDateTime),
		}
	}
	return out
}

func makeGetter(index []int, ft FieldType) func(any) any {
	return func(inst any) any {
		rv := reflect.ValueOf(inst)
		if rv.Kind() == reflect.Pointer {
			if rv.IsNil() {
				return nil
			}
			rv = rv.Elem()
		}
		field := rv.FieldByIndex(index)
		if !field.IsValid() {
			return nil
		}
		if field.Kind() == reflect.Pointer {
			if field.IsNil() {
				return nil
			}
			field = field.Elem()
		}
		val := field.Interface()
		if ft == TypeDate {
			if t, ok := val.(time.Time); ok {
				return t.Format("2006-01-02")
			}
		}
		return val
	}
}

func makeSetter(index []int, ft FieldType, destType reflect.Type) func(any, any) error {
	return func(inst any, raw any) error {
		coerced, err := coerceValue(raw, ft)
		if err != nil {
			return err
		}
		rv := reflect.ValueOf(inst)
		if rv.Kind() == reflect.Pointer {
			rv = rv.Elem()
		}
		field := rv.FieldByIndex(index)
		if !field.CanSet() {
			return fmt.Errorf("field is not settable")
		}
		// JSON null (coerced == nil) clears the dest: nil pointer, "", zero
		// time, or a nil json.RawMessage/[]byte. Missing PATCH keys never
		// reach this setter, so the update stays partial.
		assigned, err := convertTo(coerced, destType)
		if err != nil {
			return err
		}
		field.Set(assigned)
		return nil
	}
}

func convertTo(val any, dest reflect.Type) (reflect.Value, error) {
	if val == nil {
		return reflect.Zero(dest), nil
	}
	if dest.Kind() == reflect.Pointer {
		inner, err := convertTo(val, dest.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		ptr := reflect.New(dest.Elem())
		ptr.Elem().Set(inner)
		return ptr, nil
	}
	src := reflect.ValueOf(val)
	if !src.IsValid() {
		return reflect.Zero(dest), nil
	}
	if src.Type().AssignableTo(dest) {
		return src, nil
	}
	if src.Type().ConvertibleTo(dest) {
		return src.Convert(dest), nil
	}
	if dest.Kind() == reflect.String {
		return reflect.ValueOf(fmt.Sprint(val)).Convert(dest), nil
	}
	return reflect.Value{}, fmt.Errorf("cannot assign %T to %s", val, dest)
}

func makeNewInstance(elem reflect.Type) func() any {
	return func() any {
		return reflect.New(elem).Interface()
	}
}

func makeNewSlice(elem reflect.Type) func() any {
	sliceType := reflect.SliceOf(elem)
	return func() any {
		return reflect.New(sliceType).Interface()
	}
}

func makeForEach(elem reflect.Type) func(any, func(any)) {
	return func(slicePtr any, fn func(any)) {
		rv := reflect.ValueOf(slicePtr)
		if rv.Kind() == reflect.Pointer {
			rv = rv.Elem()
		}
		for i := 0; i < rv.Len(); i++ {
			item := rv.Index(i)
			if item.Kind() == reflect.Pointer {
				fn(item.Interface())
				continue
			}
			fn(item.Addr().Interface())
		}
	}
}

func elemTypeOf(model any) (reflect.Type, error) {
	if model == nil {
		return nil, fmt.Errorf("admin: nil model")
	}
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("admin: model must be a struct")
	}
	return t, nil
}

func toSnake(name string) string {
	if name == "" {
		return ""
	}
	var b strings.Builder
	runes := []rune(name)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func titleFromSlug(slug string) string {
	parts := strings.FieldsFunc(slug, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func validIdent(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func validSlug(slug string) bool {
	if slug == "" {
		return false
	}
	for i, r := range slug {
		if i == 0 {
			if r < 'a' || r > 'z' {
				return false
			}
			continue
		}
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}
