package framework

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"
)

// RequestIDHeader is the HTTP header carrying a stable per-request ID.
const RequestIDHeader = "X-Request-Id"

const requestIDGinKey = "request_id"

type requestIDContextKey struct{}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(RequestIDHeader))
		if requestID == "" {
			requestID = newRequestID()
		}
		if requestID == "" {
			requestID = "unknown"
		}

		c.Set(requestIDGinKey, requestID)
		c.Header(RequestIDHeader, requestID)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDContextKey{}, requestID))

		c.Next()
	}
}

// GetRequestID returns the request ID stored in the Gin context.
func GetRequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	value, ok := c.Get(requestIDGinKey)
	if !ok {
		return ""
	}
	requestID, _ := value.(string)
	return requestID
}

// GetRequestIDFromContext reads the request ID from a request context.
func GetRequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	buf := make([]byte, 36)
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])
	return string(buf)
}
