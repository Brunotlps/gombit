package framework

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/html"
)

const xssPasswordField = "password"

// Elements whose text content must not reach handlers (matched to HTML
// sanitizer "strict" expectations: tags stripped, dangerous element bodies
// discarded).
var xssSkipElementContent = map[string]struct{}{
	"script":   {},
	"style":    {},
	"noscript": {},
	"iframe":   {},
	"object":   {},
	"embed":    {},
	"textarea": {},
}

// xssMiddleware sanitizes HTML tags from request input before handlers run.
// JSON string values (POST/PUT/PATCH) and GET query values are stripped to
// plain text via golang.org/x/net/html. Fields named "password" are left
// unchanged.
func xssMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet:
			sanitizeQuery(c)
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			sanitizeJSONBody(c)
		}
		c.Next()
	}
}

func sanitizeQuery(c *gin.Context) {
	query := c.Request.URL.Query()
	changed := false
	for key, values := range query {
		if key == xssPasswordField {
			continue
		}
		for i, value := range values {
			cleaned := stripHTML(value)
			if cleaned != value {
				values[i] = cleaned
				changed = true
			}
		}
		query[key] = values
	}
	if changed {
		c.Request.URL.RawQuery = query.Encode()
	}
}

func sanitizeJSONBody(c *gin.Context) {
	if c.Request.Body == nil || c.Request.Body == http.NoBody {
		return
	}
	if !isJSONContentType(c.Request.Header.Get("Content-Type")) {
		return
	}

	raw, err := io.ReadAll(c.Request.Body)
	_ = c.Request.Body.Close()
	if err != nil {
		c.Request.Body = io.NopCloser(bytes.NewReader(nil))
		return
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		return
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var payload any
	if err := decoder.Decode(&payload); err != nil {
		// Leave the original body for Gin/Huma validation errors.
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		return
	}

	sanitizeJSONValue(payload, "")
	cleaned, err := json.Marshal(payload)
	if err != nil {
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(cleaned))
	c.Request.ContentLength = int64(len(cleaned))
}

func isJSONContentType(contentType string) bool {
	if contentType == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "application/json")
	}
	return strings.EqualFold(mediaType, "application/json")
}

func sanitizeJSONValue(value any, fieldName string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == xssPasswordField {
				continue
			}
			switch childValue := child.(type) {
			case string:
				typed[key] = stripHTML(childValue)
			default:
				sanitizeJSONValue(child, key)
			}
		}
	case []any:
		for i, child := range typed {
			switch childValue := child.(type) {
			case string:
				if fieldName == xssPasswordField {
					continue
				}
				typed[i] = stripHTML(childValue)
			default:
				sanitizeJSONValue(child, fieldName)
			}
		}
	}
}

// stripHTML removes HTML tags and discards content inside dangerous elements.
// Plain strings without "<" or ">" are returned unchanged.
func stripHTML(s string) string {
	if !strings.ContainsAny(s, "<>") {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))
	tokenizer := html.NewTokenizer(strings.NewReader(s))
	skipDepth := 0
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			return b.String()
		case html.TextToken:
			if skipDepth == 0 {
				b.Write(tokenizer.Text())
			}
		case html.StartTagToken:
			name, _ := tokenizer.TagName()
			if _, skip := xssSkipElementContent[string(name)]; skip {
				skipDepth++
			}
		case html.EndTagToken:
			name, _ := tokenizer.TagName()
			if _, skip := xssSkipElementContent[string(name)]; skip && skipDepth > 0 {
				skipDepth--
			}
		}
	}
}
