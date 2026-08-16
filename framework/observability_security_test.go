package framework

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/gin-gonic/gin"
)

func TestDefaultRuntimeMiddlewareOrder(t *testing.T) {
	stack := runtimeMiddlewareStack(config.Default(), newHTTPMetrics())
	got := make([]string, 0, len(stack))
	for _, middleware := range stack {
		got = append(got, middleware.name)
	}

	want := []string{
		"recovery",
		"request_id",
		"trace_context",
		"metrics",
		"security_headers",
		"xss",
		"request_timeout",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runtime middleware order = %v, want %v", got, want)
	}
}

func TestDefaultRouterAddsRequestID(t *testing.T) {
	app := newTestApp(t)
	app.Router().GET("/request-id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"gin":     GetRequestID(c),
			"context": GetRequestIDFromContext(c.Request.Context()),
		})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/request-id", nil)
	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /request-id status = %d, want %d", rec.Code, http.StatusOK)
	}
	requestID := rec.Header().Get(RequestIDHeader)
	if requestID == "" {
		t.Fatal("request ID response header is empty")
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}
	if body["gin"] != requestID || body["context"] != requestID {
		t.Fatalf("request IDs = %#v, want both values to match response header %q", body, requestID)
	}
}

func TestDefaultRouterPreservesRequestID(t *testing.T) {
	app := newTestApp(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	req.Header.Set(RequestIDHeader, "req-123")
	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /livez status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get(RequestIDHeader); got != "req-123" {
		t.Fatalf("%s = %q, want req-123", RequestIDHeader, got)
	}
}

func TestDefaultRouterCarriesTraceContext(t *testing.T) {
	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	app := newTestApp(t)
	app.Router().GET("/trace", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"gin":     GetTraceID(c),
			"context": GetTraceIDFromContext(c.Request.Context()),
		})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trace", nil)
	req.Header.Set(TraceparentHeader, "00-"+traceID+"-00f067aa0ba902b7-01")
	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /trace status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get(TraceIDHeader); got != traceID {
		t.Fatalf("%s = %q, want %q", TraceIDHeader, got, traceID)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}
	if body["gin"] != traceID || body["context"] != traceID {
		t.Fatalf("trace IDs = %#v, want both values to match traceparent trace ID %q", body, traceID)
	}
}

func TestDefaultRouterAddsSecurityHeaders(t *testing.T) {
	app := newTestApp(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /livez status = %d, want %d", rec.Code, http.StatusOK)
	}

	want := map[string]string{
		"Content-Security-Policy": "default-src 'self'",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
		"X-Content-Type-Options":  "nosniff",
		"X-Download-Options":      "noopen",
		"X-Frame-Options":         "DENY",
	}
	for header, value := range want {
		if got := rec.Header().Get(header); got != value {
			t.Fatalf("%s = %q, want %q", header, got, value)
		}
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("Strict-Transport-Security = %q, want empty outside production", got)
	}
	if got := rec.Header().Get("X-XSS-Protection"); got != "" {
		t.Fatalf("X-XSS-Protection = %q, want empty deprecated header", got)
	}
}

func TestProductionRouterAddsHSTS(t *testing.T) {
	previousMode := gin.Mode()
	t.Cleanup(func() {
		gin.SetMode(previousMode)
	})

	cfg := config.Default()
	cfg.Environment = config.EnvironmentProduction
	cfg.HTTP.Addr = "127.0.0.1:0"
	app := newTestApp(t, WithConfig(cfg))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /livez status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Strict-Transport-Security"); !strings.Contains(got, "max-age=315360000") {
		t.Fatalf("Strict-Transport-Security = %q, want max-age=315360000 in production", got)
	}
}

func TestDefaultRouterRequestTimeoutSetsDeadline(t *testing.T) {
	cfg := config.Default()
	cfg.Environment = config.EnvironmentTest
	cfg.HTTP.Addr = "127.0.0.1:0"
	cfg.HTTP.RequestTimeout = 40 * time.Millisecond

	app := newTestApp(t, WithConfig(cfg))
	app.Router().GET("/slow", func(c *gin.Context) {
		ctx := c.Request.Context()
		select {
		case <-time.After(500 * time.Millisecond):
			c.Status(http.StatusOK)
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				c.AbortWithStatus(http.StatusGatewayTimeout)
				return
			}
			c.AbortWithStatus(http.StatusInternalServerError)
		}
	})

	rec := httptest.NewRecorder()
	start := time.Now()
	app.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/slow", nil))

	if elapsed := time.Since(start); elapsed >= 200*time.Millisecond {
		t.Fatalf("GET /slow took %v, want handler to stop after request context deadline", elapsed)
	}
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("GET /slow status = %d, want %d", rec.Code, http.StatusGatewayTimeout)
	}
}

func TestRunContextConfiguresServerTimeouts(t *testing.T) {
	cfg := config.Default()
	cfg.Environment = config.EnvironmentTest
	cfg.HTTP.Addr = "127.0.0.1:0"
	cfg.HTTP.RequestTimeout = 2 * time.Second
	app := newTestApp(t, WithConfig(cfg))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- RunContext(ctx, app)
	}()
	t.Cleanup(func() {
		cancel()
		if err := waitRun(done); err != nil {
			t.Fatalf("RunContext() error = %v, want nil", err)
		}
	})

	waitForHTTP(t, app, "/livez")

	app.mu.RLock()
	server := app.server
	app.mu.RUnlock()
	if server == nil {
		t.Fatal("server = nil, want configured HTTP server")
	}
	if server.WriteTimeout != cfg.HTTP.RequestTimeout {
		t.Fatalf("server.WriteTimeout = %v, want %v", server.WriteTimeout, cfg.HTTP.RequestTimeout)
	}
	if server.IdleTimeout != cfg.HTTP.RequestTimeout {
		t.Fatalf("server.IdleTimeout = %v, want %v", server.IdleTimeout, cfg.HTTP.RequestTimeout)
	}
}

func TestDefaultRouterMetricsEndpointRecordsRequests(t *testing.T) {
	app := newTestApp(t)

	livez := httptest.NewRecorder()
	app.Router().ServeHTTP(livez, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if livez.Code != http.StatusOK {
		t.Fatalf("GET /livez status = %d, want %d", livez.Code, http.StatusOK)
	}

	metrics := httptest.NewRecorder()
	app.Router().ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if metrics.Code != http.StatusOK {
		t.Fatalf("GET /metrics status = %d, want %d", metrics.Code, http.StatusOK)
	}

	body := metrics.Body.String()
	for _, want := range []string{
		"gombit_http_active_requests",
		`gombit_http_requests_total{method="GET",route="/livez",status="200"} 1`,
		`gombit_http_request_duration_seconds_sum{method="GET",route="/livez",status="200"}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /metrics body = %q, want it to contain %q", body, want)
		}
	}
}

func TestDefaultRouterMetricsUsesBoundedLabelForUnmatchedRoutes(t *testing.T) {
	app := newTestApp(t)

	for _, path := range []string{"/missing/one", "/missing/two"} {
		rec := httptest.NewRecorder()
		app.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want %d", path, rec.Code, http.StatusNotFound)
		}
	}

	metrics := httptest.NewRecorder()
	app.Router().ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if metrics.Code != http.StatusOK {
		t.Fatalf("GET /metrics status = %d, want %d", metrics.Code, http.StatusOK)
	}

	body := metrics.Body.String()
	if !strings.Contains(body, `gombit_http_requests_total{method="GET",route="unmatched",status="404"} 2`) {
		t.Fatalf("GET /metrics body = %q, want bounded unmatched route label", body)
	}
	for _, rawPath := range []string{"/missing/one", "/missing/two"} {
		if strings.Contains(body, rawPath) {
			t.Fatalf("GET /metrics body = %q, want no raw unmatched path %q", body, rawPath)
		}
	}
}

func TestReadyzAndLivezUseEnvelope(t *testing.T) {
	app := newTestApp(t)

	for _, path := range []string{"/livez", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			app.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d, want %d", path, rec.Code, http.StatusOK)
			}
			var body struct {
				Data struct {
					Status string `json:"status"`
				} `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal %s response body: %v", path, err)
			}
			if body.Data.Status != "ok" {
				t.Fatalf("GET %s data.status = %q, want ok", path, body.Data.Status)
			}
		})
	}
}

func TestTrustedProxiesNilIgnoresForwardedFor(t *testing.T) {
	app := newTestApp(t)
	app.Router().GET("/client-ip", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/client-ip", nil)
	req.RemoteAddr = "192.0.2.10:5555"
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /client-ip status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "192.0.2.10" {
		t.Fatalf("client IP = %q, want direct peer IP", got)
	}
}

func TestTrustedProxiesInvalidConfigReturnsError(t *testing.T) {
	cfg := config.Default()
	cfg.Environment = config.EnvironmentTest
	cfg.HTTP.TrustedProxies = []string{"not-a-valid-cidr!!!"}

	_, err := New(WithConfig(cfg))
	if err == nil {
		t.Fatal("New() error = nil, want trusted proxy validation error")
	}
	if !strings.Contains(err.Error(), "trusted proxies") {
		t.Fatalf("New() error = %q, want trusted proxies message", err)
	}
}

func TestTrustedProxiesAllowsForwardedFromTrustedPeer(t *testing.T) {
	cfg := config.Default()
	cfg.Environment = config.EnvironmentTest
	cfg.HTTP.TrustedProxies = []string{"192.0.2.10/32"}
	app := newTestApp(t, WithConfig(cfg))
	app.Router().GET("/client-ip", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/client-ip", nil)
	req.RemoteAddr = "192.0.2.10:5555"
	req.Header.Set("X-Forwarded-For", "198.51.100.22")
	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /client-ip status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "198.51.100.22" {
		t.Fatalf("client IP = %q, want forwarded IP", got)
	}
}

func TestRunContextServesMetricsWithRuntimeMiddleware(t *testing.T) {
	app := newTestApp(t)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- RunContext(ctx, app)
	}()
	t.Cleanup(func() {
		cancel()
		if err := waitRun(done); err != nil {
			t.Fatalf("RunContext() error = %v, want nil", err)
		}
	})

	waitForHTTP(t, app, "/livez")
	resp := getHTTP(t, app, "/metrics")
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("read /metrics body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /metrics status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if !strings.Contains(string(body), "gombit_http_requests_total") {
		t.Fatalf("GET /metrics body = %q, want request metrics", string(body))
	}
}

func TestDefaultRouterSanitizesXSSInJSONBody(t *testing.T) {
	app := newTestApp(t)
	app.Router().POST("/comment", func(c *gin.Context) {
		var body struct {
			Comment string `json:"comment"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			t.Fatalf("bind JSON: %v", err)
		}
		c.JSON(http.StatusOK, gin.H{"comment": body.Comment})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/comment",
		strings.NewReader(`{"comment":"<script>alert(1)</script>hi"}`),
	)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /comment status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v; body: %s", err, rec.Body.String())
	}
	if body["comment"] != "hi" {
		t.Fatalf("comment = %q, want %q", body["comment"], "hi")
	}
}

func TestDefaultRouterLeavesPasswordFieldUnsanitized(t *testing.T) {
	app := newTestApp(t)
	app.Router().POST("/login", func(c *gin.Context) {
		var body struct {
			Password string `json:"password"`
			Note     string `json:"note"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			t.Fatalf("bind JSON: %v", err)
		}
		c.JSON(http.StatusOK, gin.H{
			"password": body.Password,
			"note":     body.Note,
		})
	})

	const password = `<b>secret</b>`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{"password":"`+password+`","note":"<i>hi</i>"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	app.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /login status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v; body: %s", err, rec.Body.String())
	}
	if body["password"] != password {
		t.Fatalf("password = %q, want unsanitized %q", body["password"], password)
	}
	if body["note"] != "hi" {
		t.Fatalf("note = %q, want %q", body["note"], "hi")
	}
}
