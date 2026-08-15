package framework

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/database"
	"github.com/LAA-Software-Engineering/gombit/logging"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const defaultShutdownTimeout = 10 * time.Second

// Hook is an application lifecycle callback.
type Hook func(context.Context) error

// Option configures an App.
type Option func(*App) error

// App owns Gombit's runtime lifecycle and HTTP router.
type App struct {
	cfg             config.Config
	cfgSet          bool
	db              *database.DB
	logger          *zap.Logger
	router          *gin.Engine
	startHooks      []Hook
	stopHooks       []Hook
	shutdownTimeout time.Duration

	mu     sync.RWMutex
	server *http.Server
	addr   string
}

// New creates an application using process configuration and the default router.
func New(options ...Option) (*App, error) {
	app := &App{
		cfg:             config.Default(),
		shutdownTimeout: defaultShutdownTimeout,
	}

	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(app); err != nil {
			return nil, err
		}
	}

	if !app.cfgSet {
		cfg, err := config.Load()
		if err != nil {
			return nil, err
		}
		app.cfg = cfg
	}

	if err := app.cfg.Validate(); err != nil {
		return nil, err
	}
	if app.shutdownTimeout <= 0 {
		return nil, errors.New("framework: shutdown timeout must be positive")
	}
	configureHTTPMode(app.cfg)
	if app.logger == nil {
		logger, err := logging.New(app.cfg.Logging)
		if err != nil {
			return nil, err
		}
		app.logger = logger
	}
	if app.router == nil {
		app.router = newRouter()
	}

	return app, nil
}

// WithConfig sets the app configuration.
func WithConfig(cfg config.Config) Option {
	return func(app *App) error {
		if err := cfg.Validate(); err != nil {
			return err
		}
		app.cfg = cfg
		app.cfgSet = true
		return nil
	}
}

// WithDatabase attaches an opened database handle to the app.
func WithDatabase(db *database.DB) Option {
	return func(app *App) error {
		if db == nil || db.DB == nil {
			return errors.New("framework: nil database")
		}
		app.db = db
		return nil
	}
}

// WithLogger attaches a Zap logger to the app.
func WithLogger(logger *zap.Logger) Option {
	return func(app *App) error {
		if logger == nil {
			return errors.New("framework: nil logger")
		}
		app.logger = logger
		return nil
	}
}

// WithRouter sets the app router.
func WithRouter(router *gin.Engine) Option {
	return func(app *App) error {
		if router == nil {
			return errors.New("framework: nil router")
		}
		app.router = router
		return nil
	}
}

// WithShutdownTimeout sets the bounded shutdown timeout.
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(app *App) error {
		if timeout <= 0 {
			return errors.New("framework: shutdown timeout must be positive")
		}
		app.shutdownTimeout = timeout
		return nil
	}
}

// Config returns the typed app configuration.
func (a *App) Config() config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

// Logger returns the app's Zap logger.
func (a *App) Logger() *zap.Logger {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.logger
}

// Database returns the opened database handle with driver metadata.
func (a *App) Database() *database.DB {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.db
}

// DB returns the underlying GORM database escape hatch.
func (a *App) DB() *gorm.DB {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.db == nil {
		return nil
	}
	return a.db.DB
}

// Router returns the underlying Gin router escape hatch.
func (a *App) Router() *gin.Engine {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.router
}

// Addr returns the bound HTTP address after the app has started.
func (a *App) Addr() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.addr
}

// OnStart registers a start hook. Hooks run in registration order.
func (a *App) OnStart(hook Hook) {
	if hook == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.startHooks = append(a.startHooks, hook)
}

// OnStop registers a stop hook. Hooks run in reverse registration order.
func (a *App) OnStop(hook Hook) {
	if hook == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.stopHooks = append(a.stopHooks, hook)
}

// Run runs app until an interrupt or terminate signal is received.
func Run(app *App) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return RunContext(ctx, app)
}

// RunContext runs app until ctx is canceled or the HTTP server fails.
func RunContext(ctx context.Context, app *App) error {
	if ctx == nil {
		return errors.New("framework: nil context")
	}
	if app == nil {
		return errors.New("framework: nil app")
	}

	listener, err := net.Listen("tcp", app.Config().HTTP.Addr)
	if err != nil {
		return fmt.Errorf("framework: listen: %w", err)
	}

	server := &http.Server{
		Handler:           app.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	app.setServer(server, listener.Addr().String())

	if err := app.runStartHooks(ctx); err != nil {
		_ = listener.Close()
		return errors.Join(err, app.runStopHooks())
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case <-ctx.Done():
		if err := app.shutdown(); err != nil {
			return err
		}
		if err := <-serverErr; err != nil {
			return fmt.Errorf("framework: serve: %w", err)
		}
		return nil
	case err := <-serverErr:
		stopErr := app.runStopHooks()
		if err != nil {
			return errors.Join(fmt.Errorf("framework: serve: %w", err), stopErr)
		}
		return stopErr
	}
}

func (a *App) setServer(server *http.Server, addr string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.server = server
	a.addr = addr
}

func (a *App) snapshotStartHooks() []Hook {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]Hook(nil), a.startHooks...)
}

func (a *App) snapshotStopHooks() []Hook {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]Hook(nil), a.stopHooks...)
}

func (a *App) runStartHooks(ctx context.Context) error {
	for i, hook := range a.snapshotStartHooks() {
		if err := hook(ctx); err != nil {
			return fmt.Errorf("framework: start hook %d: %w", i+1, err)
		}
	}
	return nil
}

func (a *App) shutdown() error {
	a.mu.RLock()
	server := a.server
	timeout := a.shutdownTimeout
	a.mu.RUnlock()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if server != nil {
		if err := server.Shutdown(shutdownCtx); err != nil {
			return errors.Join(
				fmt.Errorf("framework: shutdown: %w", err),
				a.runStopHooksWithContext(shutdownCtx),
			)
		}
	}

	return a.runStopHooksWithContext(shutdownCtx)
}

func (a *App) runStopHooks() error {
	a.mu.RLock()
	timeout := a.shutdownTimeout
	a.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return a.runStopHooksWithContext(ctx)
}

func (a *App) runStopHooksWithContext(ctx context.Context) error {
	hooks := a.snapshotStopHooks()
	var joined error
	for i := len(hooks) - 1; i >= 0; i-- {
		if err := hooks[i](ctx); err != nil {
			joined = errors.Join(joined, fmt.Errorf("framework: stop hook %d: %w", i+1, err))
		}
	}
	return joined
}

func newRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"status": "ok",
			},
		})
	})
	router.GET("/readyz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"status": "ok",
			},
		})
	})
	return router
}

func configureHTTPMode(cfg config.Config) {
	if cfg.Environment == config.EnvironmentProduction {
		gin.SetMode(gin.ReleaseMode)
	}
}
