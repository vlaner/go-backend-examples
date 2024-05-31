package main

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var requestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "backend",
		Name:      "http_request_duration_seconds",
		Help:      "Request duration in seconds",
		Buckets:   prometheus.DefBuckets,
	},
	[]string{"endpoint", "method"},
)

var requestDurationSummary = prometheus.NewSummary(
	prometheus.SummaryOpts{
		Namespace: "backend",
		Name:      "http_summary_request_duration_seconds",
		Help:      "Request duration in seconds using summary",
		// use p50, P90 and P99 targets
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	},
)

func MetricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		requestTime := time.Since(start).Seconds()

		requestDuration.With(prometheus.Labels{"endpoint": r.URL.Path, "method": r.Method}).Observe(requestTime)
		requestDurationSummary.Observe(requestTime)
	})
}

func Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	randomDuration := time.Duration(rand.Intn(1000)+100) * time.Millisecond
	time.Sleep(randomDuration)
	w.Write([]byte("hello"))
}

func appServer() *http.Server {
	appMux := http.NewServeMux()
	appMux.Handle("GET /", MetricsMiddleware(Index))

	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      appMux,
	}

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
		log.Println("Stopped app server")
	}()

	return server
}

func metricsServer() *http.Server {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:         ":8081",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      metricsMux,
	}

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
		log.Println("Stopped metrics server")
	}()

	return server
}

func main() {
	prometheus.MustRegister(requestDuration, requestDurationSummary)

	appSrv := appServer()
	metricsSrv := metricsServer()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := appSrv.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("HTTP shutdown error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("HTTP shutdown error: %v", err)
		}
	}()

	wg.Wait()
	log.Println("Graceful shutdown complete")
}
