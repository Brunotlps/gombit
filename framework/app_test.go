package framework

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/gin-gonic/gin"
)

func TestRunContextBootsAndShutsDown(t *testing.T) {
	app := newTestApp(t)
	app.Router().GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]any{
			"data": map[string]string{"status": "ok"},
		})
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- RunContext(ctx, app)
	}()

	waitForHTTP(t, app, "/livez")
	waitForHTTP(t, app, "/ping")
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunContext() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunContext() did not shut down after context cancellation")
	}
}

func TestHooksRunInDeterministicOrder(t *testing.T) {
	app := newTestApp(t)
	var (
		mu    sync.Mutex
		order []string
	)
	record := func(value string) Hook {
		return func(ctx context.Context) error {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, value)
			return nil
		}
	}

	app.OnStart(record("start-1"))
	app.OnStart(record("start-2"))
	app.OnStop(record("stop-1"))
	app.OnStop(record("stop-2"))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- RunContext(ctx, app)
	}()

	waitForHTTP(t, app, "/readyz")
	cancel()
	if err := waitRun(done); err != nil {
		t.Fatalf("RunContext() error = %v, want nil", err)
	}

	want := []string{"start-1", "start-2", "stop-2", "stop-1"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("hook order = %#v, want %#v", order, want)
	}
}

func TestStartFailureRunsStopHooks(t *testing.T) {
	app := newTestApp(t)
	startErr := errors.New("start failed")
	var order []string

	app.OnStart(func(ctx context.Context) error {
		order = append(order, "start-1")
		return nil
	})
	app.OnStart(func(ctx context.Context) error {
		order = append(order, "start-2")
		return startErr
	})
	app.OnStop(func(ctx context.Context) error {
		order = append(order, "stop-1")
		return nil
	})
	app.OnStop(func(ctx context.Context) error {
		order = append(order, "stop-2")
		return nil
	})

	err := RunContext(context.Background(), app)
	if !errors.Is(err, startErr) {
		t.Fatalf("RunContext() error = %v, want %v", err, startErr)
	}

	want := []string{"start-1", "start-2", "stop-2", "stop-1"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("hook order = %#v, want %#v", order, want)
	}
}

func TestStopHooksReceiveBoundedShutdownContext(t *testing.T) {
	app := newTestApp(t, WithShutdownTimeout(250*time.Millisecond))
	deadlineSeen := make(chan bool, 1)
	app.OnStop(func(ctx context.Context) error {
		_, ok := ctx.Deadline()
		deadlineSeen <- ok
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- RunContext(ctx, app)
	}()

	waitForHTTP(t, app, "/livez")
	cancel()
	if err := waitRun(done); err != nil {
		t.Fatalf("RunContext() error = %v, want nil", err)
	}

	select {
	case got := <-deadlineSeen:
		if !got {
			t.Fatal("stop hook context had no deadline")
		}
	default:
		t.Fatal("stop hook did not run")
	}
}

func TestNewValidatesOptions(t *testing.T) {
	_, err := New(WithShutdownTimeout(0))
	if err == nil {
		t.Fatal("New(WithShutdownTimeout(0)) error = nil, want error")
	}
}

func newTestApp(t *testing.T, options ...Option) *App {
	t.Helper()

	cfg := config.Default()
	cfg.Environment = config.EnvironmentTest
	cfg.HTTP.Addr = "127.0.0.1:0"

	options = append([]Option{WithConfig(cfg)}, options...)
	app, err := New(options...)
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	return app
}

func waitForHTTP(t *testing.T, app *App, path string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		addr := app.Addr()
		if addr == "" {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		resp, err := http.Get("http://" + addr + path)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s on %s", path, app.Addr())
}

func waitRun(done <-chan error) error {
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		return errors.New("RunContext did not return")
	}
}
