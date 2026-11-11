package lifecycle

import (
	"context"
	"errors"
	"slices"
	"sync"
)

var ErrShutdownTimeout = errors.New("lifecycle: shutdown timeout")

type App struct {
	ctx    context.Context
	cancel context.CancelCauseFunc
	mu     sync.Mutex
	runs   []func(context.Context) error
	errs   []error
}

func New(ctx context.Context) *App {
	app := &App{}
	app.ctx, app.cancel = context.WithCancelCause(ctx)

	return app
}

func (a *App) Context() context.Context {
	return a.ctx
}

func (a *App) Add(runs ...func(context.Context) error) {
	a.mu.Lock()
	a.runs = append(a.runs, runs...)
	a.mu.Unlock()
}

func (a *App) Shutdown() {
	a.cancel(nil)
}

func (a *App) Wait() error {
	runs := a.pendingRuns()

	var wg sync.WaitGroup
	for _, run := range runs {
		wg.Go(func() {
			err := run(a.ctx)
			if err == nil || errors.Is(err, context.Canceled) {
				return
			}
			a.fail(err)
		})
	}
	wg.Wait()

	return a.joinedErrors()
}

func (a *App) pendingRuns() []func(context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	return slices.Clone(a.runs)
}

func (a *App) fail(err error) {
	a.mu.Lock()
	a.errs = append(a.errs, err)
	a.mu.Unlock()

	a.cancel(err)
}

func (a *App) joinedErrors() error {
	errs := a.errors()
	cause := context.Cause(a.ctx)
	if cause != nil &&
		!errors.Is(cause, context.Canceled) &&
		!slices.ContainsFunc(errs, func(err error) bool { return errors.Is(err, cause) }) {
		errs = append(errs, cause)
	}

	return errors.Join(errs...)
}

func (a *App) errors() []error {
	a.mu.Lock()
	defer a.mu.Unlock()

	return slices.Clone(a.errs)
}
