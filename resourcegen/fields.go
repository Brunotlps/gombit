package resourcegen

import (
	"fmt"
	"strings"
)

// Field is one parsed resource field from the CLI grammar
// name:type[:required][,unique][,index] (design §27 subset).
type Field struct {
	Name     string
	JSONName string
	GoName   string
	Type     FieldType
	GoType   string
	Required bool
	Unique   bool
	Index    bool
	Nullable bool
}

// FieldType is a supported scalar in the v0.1 subset.
type FieldType string

const (
	FieldString FieldType = "string"
	FieldText   FieldType = "text"
	FieldInt    FieldType = "int"
	FieldInt64  FieldType = "int64"
	FieldBool   FieldType = "bool"
	FieldUint   FieldType = "uint"
)

var supportedTypes = []FieldType{
	FieldString, FieldText, FieldInt, FieldInt64, FieldBool, FieldUint,
}

func parseFields(specs []string) ([]Field, error) {
	seen := make(map[string]struct{}, len(specs))
	fields := make([]Field, 0, len(specs))
	for _, spec := range specs {
		field, err := parseField(spec)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(field.JSONName)
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("resourcegen: duplicate field %q", field.JSONName)
		}
		seen[key] = struct{}{}
		fields = append(fields, field)
	}
	return fields, nil
}

func parseField(spec string) (Field, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return Field{}, fmt.Errorf("resourcegen: empty field spec")
	}
	parts := strings.Split(spec, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return Field{}, fmt.Errorf("resourcegen: field %q must be name:type[:modifiers]", spec)
	}
	name := strings.TrimSpace(parts[0])
	typeName := strings.ToLower(strings.TrimSpace(parts[1]))
	if name == "" || typeName == "" {
		return Field{}, fmt.Errorf("resourcegen: field %q must be name:type[:modifiers]", spec)
	}

	jsonName := toSnake(name)
	goName := toPascal(name)
	if jsonName == "" || !isExportedIdent(goName) {
		return Field{}, fmt.Errorf("resourcegen: field name %q is not a valid identifier", name)
	}
	if _, reserved := reservedFields[jsonName]; reserved {
		return Field{}, fmt.Errorf("resourcegen: field %q conflicts with gorm.Model", jsonName)
	}

	goType, fieldType, err := mapFieldType(typeName)
	if err != nil {
		return Field{}, err
	}

	field := Field{
		Name:     name,
		JSONName: jsonName,
		GoName:   goName,
		Type:     fieldType,
		GoType:   goType,
	}
	if len(parts) == 3 {
		if err := applyModifiers(&field, parts[2]); err != nil {
			return Field{}, err
		}
	}
	return field, nil
}

func mapFieldType(typeName string) (string, FieldType, error) {
	switch typeName {
	case "string":
		return "string", FieldString, nil
	case "text":
		return "string", FieldText, nil
	case "int":
		return "int", FieldInt, nil
	case "int64":
		return "int64", FieldInt64, nil
	case "bool":
		return "bool", FieldBool, nil
	case "uint":
		return "uint", FieldUint, nil
	default:
		names := make([]string, 0, len(supportedTypes))
		for _, item := range supportedTypes {
			names = append(names, string(item))
		}
		return "", "", fmt.Errorf("resourcegen: unknown type %q (supported: %s)", typeName, strings.Join(names, ", "))
	}
}

func applyModifiers(field *Field, raw string) error {
	for _, part := range strings.Split(raw, ",") {
		mod := strings.ToLower(strings.TrimSpace(part))
		if mod == "" {
			continue
		}
		switch {
		case mod == "required":
			field.Required = true
		case mod == "unique":
			field.Unique = true
		case mod == "index":
			field.Index = true
		case mod == "nullable":
			field.Nullable = true
		case strings.HasPrefix(mod, "default="), strings.HasPrefix(mod, "min="), strings.HasPrefix(mod, "max="), strings.HasPrefix(mod, "references="):
			return fmt.Errorf("resourcegen: modifier %q is not supported in this milestone (supported: required, unique, index, nullable)", mod)
		default:
			return fmt.Errorf("resourcegen: unknown modifier %q (supported: required, unique, index, nullable)", mod)
		}
	}
	if field.Required && field.Nullable {
		return fmt.Errorf("resourcegen: field %q cannot be both required and nullable", field.JSONName)
	}
	return nil
}

func (f Field) gormTag() string {
	var parts []string
	switch f.Type {
	case FieldString:
		parts = append(parts, "size:255")
	case FieldText:
		parts = append(parts, "type:text")
	}
	if f.Required && !f.Nullable {
		parts = append(parts, "not null")
	}
	switch {
	case f.Unique:
		parts = append(parts, "uniqueIndex")
	case f.Index:
		parts = append(parts, "index")
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ";")
}

func (f Field) humaTags() string {
	var parts []string
	parts = append(parts, fmt.Sprintf(`json:"%s"`, f.JSONName))
	switch f.Type {
	case FieldString:
		if f.Required {
			parts = append(parts, `minLength:"1"`)
		}
		parts = append(parts, `maxLength:"255"`)
	case FieldText:
		if f.Required {
			parts = append(parts, `minLength:"1"`)
		}
	case FieldUint:
		parts = append(parts, `minimum:"0"`)
	}
	parts = append(parts, fmt.Sprintf(`doc:"%s"`, f.GoName))
	return strings.Join(parts, " ")
}

func (f Field) jsonTag() string {
	return fmt.Sprintf(`json:"%s"`, f.JSONName)
}

func (f Field) inputKind() string {
	switch f.Type {
	case FieldText:
		return "textarea"
	case FieldBool:
		return "checkbox"
	case FieldInt, FieldInt64, FieldUint:
		return "number"
	default:
		return "text"
	}
}
