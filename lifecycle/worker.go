package lifecycle

import (
	"context"
	"sync"
	"time"
)

const (
	defaultWorkerConcurrency     = 4
	defaultWorkerShutdownTimeout = 5 * time.Second
)

type Job func(context.Context)

type WorkerOption func(*Worker)

func WithWorkerConcurrency(concurrency int) WorkerOption {
	return func(worker *Worker) {
		if concurrency > 0 {
			worker.concurrency = concurrency
		}
	}
}

func WithWorkerShutdownTimeout(timeout time.Duration) WorkerOption {
	return func(worker *Worker) {
		if timeout > 0 {
			worker.shutdownTimeout = timeout
		}
	}
}

type Worker struct {
	jobs            <-chan Job
	concurrency     int
	shutdownTimeout time.Duration
}

func NewWorker(jobs <-chan Job, opts ...WorkerOption) *Worker {
	worker := &Worker{
		jobs:            jobs,
		concurrency:     defaultWorkerConcurrency,
		shutdownTimeout: defaultWorkerShutdownTimeout,
	}
	for _, opt := range opts {
		opt(worker)
	}

	return worker
}

func (w *Worker) Run(ctx context.Context) error {
	workCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()

	var wg sync.WaitGroup
	for range w.concurrency {
		wg.Go(func() {
			for {
				if ctx.Err() != nil {
					return
				}

				select {
				case job, ok := <-w.jobs:
					if !ok {
						return
					}
					job(workCtx)
				case <-ctx.Done():
					return
				}
			}
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.WithoutCancel(ctx), w.shutdownTimeout)
	defer cancelShutdown()

	select {
	case <-done:
		return nil
	case <-shutdownCtx.Done():
		cancel()
		return ErrShutdownTimeout
	}
}
