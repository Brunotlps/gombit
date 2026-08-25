package contract

import (
	"regexp"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

var requiredPropertyRE = regexp.MustCompile(`(?i)^expected required property (\S+) to be present$`)

// FieldsFromErrors maps Huma validation detail errors into D10 field errors.
//
// Body locations lose the leading "body." prefix so JSON field names match the
// request body. Query, path, and header locations keep their prefixes.
func FieldsFromErrors(errs ...error) map[string][]string {
	if len(errs) == 0 {
		return nil
	}

	fields := make(map[string][]string)
	for _, err := range errs {
		if err == nil {
			continue
		}
		key, message := fieldFromError(err)
		if key == "" || message == "" {
			continue
		}
		fields[key] = append(fields[key], message)
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func fieldFromError(err error) (key, message string) {
	if detailer, ok := err.(huma.ErrorDetailer); ok {
		detail := detailer.ErrorDetail()
		if detail == nil {
			return "", ""
		}
		message = strings.TrimSpace(detail.Message)
		if message == "" {
			message = strings.TrimSpace(detail.Error())
		}
		key = fieldKey(detail.Location)
		if key == "_error" {
			if inferred := fieldFromMessage(message); inferred != "" {
				key = inferred
			}
		}
		return key, message
	}

	message = strings.TrimSpace(err.Error())
	if message == "" {
		return "", ""
	}
	if inferred := fieldFromMessage(message); inferred != "" {
		return inferred, message
	}
	// Generic errors are not field-level validation details. Huma's
	// unexpected-500 path wraps the handler error here; putting it in
	// fields._error would classify the response as validation_error.
	return "", ""
}

func fieldKey(location string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return "_error"
	}
	if strings.HasPrefix(location, "body.") {
		return strings.TrimPrefix(location, "body.")
	}
	if location == "body" {
		return "_error"
	}
	return location
}

func fieldFromMessage(message string) string {
	matches := requiredPropertyRE.FindStringSubmatch(strings.TrimSpace(message))
	if len(matches) != 2 {
		return ""
	}
	return matches[1]
}
