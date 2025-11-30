package worker

import (
	"context"
	"sync"

	"github.com/Sokol111/ecommerce-commons/pkg/core/health"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// worker represents a background worker that can be started and stopped.
type worker interface {
	Start()
	Stop()
}

// runnable is a type that has a Run method that can return a fatal error.
type runnable interface {
	Run(ctx context.Context) error
}

// Options contains configuration for a worker.
type Options struct {
	WaitForTrafficReady bool
	WaitReady           bool
	ShutdownOnError     bool
}

// Option is a functional option for configuring a worker.
type Option func(*Options)

// WithTrafficReady makes the worker wait for traffic readiness before starting.
func WithTrafficReady() Option {
	return func(o *Options) {
		o.WaitForTrafficReady = true
	}
}

// WithReady makes the worker wait for all components to be ready before starting.
func WithReady() Option {
	return func(o *Options) {
		o.WaitReady = true
	}
}

// WithShutdown makes the worker trigger application shutdown on fatal error.
func WithShutdown() Option {
	return func(o *Options) {
		o.ShutdownOnError = true
	}
}

// BaseWorker is a universal worker implementation.
type baseWorker struct {
	name       string
	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
	log        *zap.Logger
	runFunc    func(ctx context.Context) error
	shutdowner fx.Shutdowner
	readiness  health.ReadinessWaiter
	options    Options
}

// Start starts the worker by running the function in a goroutine.
func (w *baseWorker) Start() {
	w.log.Info("starting " + w.name)
	w.ctx, w.cancelFunc = context.WithCancel(context.Background())
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.run()
	}()
}

func (w *baseWorker) run() {
	// Wait for components readiness if configured
	if w.options.WaitReady {
		w.log.Info("waiting for components readiness")
		if err := w.readiness.WaitReady(w.ctx); err != nil {
			w.log.Info(w.name + " stopped (cancelled while waiting for readiness)")
			return
		}
		w.log.Info("components readiness achieved")
	}

	// Wait for traffic readiness if configured
	if w.options.WaitForTrafficReady {
		w.log.Info("waiting for traffic readiness")
		if err := w.readiness.WaitForTrafficReady(w.ctx); err != nil {
			w.log.Info(w.name + " stopped (cancelled while waiting for traffic readiness)")
			return
		}
		w.log.Info("traffic readiness achieved, starting work")
	}

	// Run the main function
	err := w.runFunc(w.ctx)
	if err == nil {
		w.log.Info(w.name + " stopped")
		return
	}

	// Handle error
	if w.options.ShutdownOnError {
		w.log.Error(w.name+" fatal error, initiating shutdown", zap.Error(err))
		if shutdownErr := w.shutdowner.Shutdown(fx.ExitCode(1)); shutdownErr != nil {
			w.log.Error("failed to initiate shutdown", zap.Error(shutdownErr))
		}
	} else {
		w.log.Error(w.name+" stopped with error", zap.Error(err))
	}
}

// Stop stops the worker by canceling the context and waiting for the goroutine to finish.
func (w *baseWorker) Stop() {
	w.log.Info("stopping " + w.name)
	if w.cancelFunc != nil {
		w.cancelFunc()
	}
	w.wg.Wait()
}

// registerWorker registers a worker with fx.Lifecycle to start and stop with the application.
func registerWorker(lc fx.Lifecycle, w worker) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			w.Start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			w.Stop()
			return nil
		},
	})
}

// Register creates an fx.Annotate that provides a worker for the given dependency type.
// The dependency must have a Run(ctx context.Context) error method.
//
// Options:
//   - WithReady(): wait for all components to be ready before starting
//   - WithTrafficReady(): wait for traffic readiness before starting
//   - WithShutdown(): trigger application shutdown on fatal error
//
// Example:
//
//	worker.Register[*reader]("reader", worker.WithTrafficReady(), worker.WithShutdown())
//	worker.Register[*processor]("processor", worker.WithReady())
func Register[T runnable](name string, opts ...Option) any {
	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return fx.Annotate(
		func(lc fx.Lifecycle, log *zap.Logger, shutdowner fx.Shutdowner, readiness health.ReadinessWaiter, dep T) worker {
			w := &baseWorker{
				name:       name,
				log:        log,
				runFunc:    dep.Run,
				shutdowner: shutdowner,
				readiness:  readiness,
				options:    options,
			}
			registerWorker(lc, w)
			return w
		},
		fx.ResultTags(`group:"workers"`),
	)
}
