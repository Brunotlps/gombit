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

// requestMeta carries the per-request correlation IDs propagated through the
// request context by requestContextMiddleware. Keeping both under one context
// key costs a single context.WithValue and a single Request.WithContext per
// request; the previously separate request_id and trace_context middlewares
// each ran their own pair.
type requestMeta struct {
	requestID string
	traceID   string
}

type requestMetaKey struct{}

// requestContextMiddleware assigns the request and trace correlation IDs
// (honoring an inbound X-Request-Id and W3C traceparent when present), exposes
// them on the response headers and the Gin context, and propagates both
// through the request context under a single key. It replaces the previously
// separate request_id and trace_context middlewares: splitting them cost an
// extra context.WithValue and Request.WithContext allocation per request for
// no behavioral difference (the two IDs are independent and both must land
// before the metrics/handler stages either way).
func requestContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(RequestIDHeader))
		if requestID == "" {
			requestID = newRequestID()
		}
		if requestID == "" {
			requestID = "unknown"
		}

		traceID := traceIDFromTraceparent(c.GetHeader(TraceparentHeader))
		if traceID == "" {
			traceID = newTraceID()
		}
		if traceID == "" {
			traceID = "unknown"
		}

		c.Set(requestIDGinKey, requestID)
		c.Set(traceIDGinKey, traceID)
		c.Header(RequestIDHeader, requestID)
		c.Header(TraceIDHeader, traceID)
		c.Request = c.Request.WithContext(
			context.WithValue(c.Request.Context(), requestMetaKey{}, requestMeta{
				requestID: requestID,
				traceID:   traceID,
			}),
		)

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
	meta, _ := ctx.Value(requestMetaKey{}).(requestMeta)
	return meta.requestID
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
