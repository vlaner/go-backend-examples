package lifecycle_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vlaner/go-backend-examples/lifecycle"
)

const (
	requestTimeout          = 5 * time.Second
	gracefulShutdownTimeout = 5 * time.Second
	shortShutdownTimeout    = 100 * time.Millisecond
	loadRequestCount        = 200
	loadConcurrency         = 20
	workerConcurrency       = 8
)

func TestLifecycleHandlesManyRequestsAndJobs(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	app := lifecycle.New(ctx)
	t.Cleanup(app.Shutdown)

	listener := newListener(t)
	jobs := make(chan lifecycle.Job, loadRequestCount)
	worker := lifecycle.NewWorker(
		jobs,
		lifecycle.WithWorkerConcurrency(workerConcurrency),
		lifecycle.WithWorkerShutdownTimeout(gracefulShutdownTimeout),
	)
	app.Add(worker.Run)

	var processed atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		done := make(chan struct{})
		jobs <- func(_ context.Context) {
			processed.Add(1)
			close(done)
		}
		<-done
		w.WriteHeader(http.StatusOK)
	})
	server := lifecycle.NewHTTPServer(
		&http.Server{Handler: handler, ReadHeaderTimeout: gracefulShutdownTimeout},
		lifecycle.WithHTTPServerListener(listener),
		lifecycle.WithHTTPServerShutdownTimeout(gracefulShutdownTimeout),
	)
	app.Add(server.Run)

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- app.Wait()
	}()

	client := &http.Client{Timeout: requestTimeout}
	url := "http://" + listener.Addr().String()

	var failed atomic.Int64
	var wg sync.WaitGroup
	for range loadConcurrency {
		wg.Go(func() {
			for range loadRequestCount / loadConcurrency {
				err := get(client, url, http.StatusOK)
				if err != nil {
					failed.Add(1)
				}
			}
		})
	}
	wg.Wait()
	client.CloseIdleConnections()

	if failed.Load() != 0 {
		t.Fatalf("%d of %d requests failed", failed.Load(), loadRequestCount)
	}
	if processed.Load() != loadRequestCount {
		t.Fatalf("processed %d jobs, want %d", processed.Load(), loadRequestCount)
	}

	app.Shutdown()

	err := <-waitDone
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
}

func TestLifecycleDrainsInflightRequestsAndJobs(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	app := lifecycle.New(ctx)
	t.Cleanup(app.Shutdown)

	listener := newListener(t)
	jobs := make(chan lifecycle.Job, workerConcurrency)
	worker := lifecycle.NewWorker(
		jobs,
		lifecycle.WithWorkerConcurrency(workerConcurrency),
		lifecycle.WithWorkerShutdownTimeout(gracefulShutdownTimeout),
	)
	app.Add(worker.Run)

	started := make(chan struct{}, workerConcurrency)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseFunc := func() {
		releaseOnce.Do(func() {
			close(release)
		})
	}
	t.Cleanup(releaseFunc)

	var jobsDone sync.WaitGroup
	jobsDone.Add(workerConcurrency)
	jobsFinished := make(chan struct{})
	go func() {
		jobsDone.Wait()
		close(jobsFinished)
	}()

	var handlersDone sync.WaitGroup
	handlersDone.Add(workerConcurrency)
	handlersFinished := make(chan struct{})
	go func() {
		handlersDone.Wait()
		close(handlersFinished)
	}()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		defer handlersDone.Done()
		done := make(chan struct{})
		jobs <- func(_ context.Context) {
			defer jobsDone.Done()
			started <- struct{}{}
			<-release
			close(done)
		}
		<-done
		w.WriteHeader(http.StatusNoContent)
	})
	server := lifecycle.NewHTTPServer(
		&http.Server{Handler: handler, ReadHeaderTimeout: gracefulShutdownTimeout},
		lifecycle.WithHTTPServerListener(listener),
		lifecycle.WithHTTPServerShutdownTimeout(gracefulShutdownTimeout),
	)
	app.Add(server.Run)

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- app.Wait()
	}()

	client := &http.Client{Timeout: requestTimeout}
	url := "http://" + listener.Addr().String()
	results := make(chan error, workerConcurrency)
	for range workerConcurrency {
		go func() {
			results <- get(client, url, http.StatusNoContent)
		}()
	}
	for range workerConcurrency {
		<-started
	}

	app.Shutdown()
	waitForDialFailure(t, listener.Addr().String())

	select {
	case err := <-waitDone:
		t.Fatalf("Wait() returned before release: %v", err)
	default:
	}

	releaseFunc()
	jobsDone.Wait()

	for range workerConcurrency {
		err := <-results
		if err != nil {
			t.Fatalf("request error = %v", err)
		}
	}

	err := <-waitDone
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
}

func TestLifecycleDropsHangingRequestsAndJobs(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	app := lifecycle.New(ctx)
	t.Cleanup(app.Shutdown)

	listener := newListener(t)
	jobs := make(chan lifecycle.Job, workerConcurrency)
	worker := lifecycle.NewWorker(
		jobs,
		lifecycle.WithWorkerConcurrency(workerConcurrency),
		lifecycle.WithWorkerShutdownTimeout(shortShutdownTimeout),
	)
	app.Add(worker.Run)

	started := make(chan struct{}, workerConcurrency)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseFunc := func() {
		releaseOnce.Do(func() {
			close(release)
		})
	}
	t.Cleanup(releaseFunc)

	var jobsDone sync.WaitGroup
	jobsDone.Add(workerConcurrency)
	jobsFinished := make(chan struct{})
	go func() {
		jobsDone.Wait()
		close(jobsFinished)
	}()

	var handlersDone sync.WaitGroup
	handlersDone.Add(workerConcurrency)
	handlersFinished := make(chan struct{})
	go func() {
		handlersDone.Wait()
		close(handlersFinished)
	}()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		defer handlersDone.Done()
		done := make(chan struct{})
		jobs <- func(_ context.Context) {
			defer jobsDone.Done()
			started <- struct{}{}
			<-release
			close(done)
		}
		<-done
		w.WriteHeader(http.StatusNoContent)
	})
	server := lifecycle.NewHTTPServer(
		&http.Server{Handler: handler, ReadHeaderTimeout: gracefulShutdownTimeout},
		lifecycle.WithHTTPServerListener(listener),
		lifecycle.WithHTTPServerShutdownTimeout(shortShutdownTimeout),
	)
	app.Add(server.Run)

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- app.Wait()
	}()

	client := &http.Client{Timeout: requestTimeout}
	url := "http://" + listener.Addr().String()
	results := make(chan error, workerConcurrency)
	for range workerConcurrency {
		go func() {
			results <- get(client, url, http.StatusNoContent)
		}()
	}
	for range workerConcurrency {
		<-started
	}

	app.Shutdown()

	err := <-waitDone
	if !errors.Is(err, lifecycle.ErrShutdownTimeout) {
		t.Fatalf("Wait() error = %v, want %v", err, lifecycle.ErrShutdownTimeout)
	}

	select {
	case <-jobsFinished:
		t.Fatal("hanging jobs finished before release")
	default:
	}
	select {
	case <-handlersFinished:
		t.Fatal("hanging requests finished before release")
	default:
	}

	for range workerConcurrency {
		result := <-results
		if result == nil {
			t.Fatal("hanging request succeeded instead of being dropped")
		}
	}

	waitForDialFailure(t, listener.Addr().String())

	releaseFunc()
	jobsDone.Wait()
	handlersDone.Wait()
}

func waitForDialFailure(t *testing.T, address string) {
	t.Helper()

	deadline := time.Now().Add(gracefulShutdownTimeout)
	for {
		conn, err := net.DialTimeout("tcp", address, shortShutdownTimeout)
		if err != nil {
			return
		}
		_ = conn.Close()
		if time.Until(deadline) <= 0 {
			t.Fatal("server continued accepting connections during shutdown")
		}
	}
}

func newListener(t *testing.T) net.Listener {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	return listener
}

func get(client *http.Client, url string, wantStatus int) error {
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != wantStatus {
		return fmt.Errorf("status = %d, want %d", response.StatusCode, wantStatus)
	}

	return nil
}
