package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var requestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "backend",
		Name:      "http_request_duration_seconds",
		Help:      "Request duration in seconds",
		Buckets:   prometheus.DefBuckets,
	},
	[]string{"endpoint", "status", "method"},
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

var requestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "backend",
		Name:      "http_requests_total",
	},
	[]string{"endpoint", "status", "method"},
)

func MetricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := NewStatusResponseWriter(w)
		start := time.Now()
		next.ServeHTTP(rw, r)
		requestTime := time.Since(start).Seconds()

		requestDuration.WithLabelValues(r.URL.Path, rw.GetStatusString(), r.Method).Observe(requestTime)
		requestDurationSummary.Observe(requestTime)
		requestsTotal.WithLabelValues(r.URL.Path, rw.GetStatusString(), r.Method).Inc()
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

type StatusResponseWriter struct {
	http.ResponseWriter
	status int
}

func NewStatusResponseWriter(w http.ResponseWriter) *StatusResponseWriter {
	return &StatusResponseWriter{w, http.StatusOK}
}

func (srw *StatusResponseWriter) WriteHeader(status int) {
	srw.status = status
	srw.ResponseWriter.WriteHeader(status)
}

func (srw *StatusResponseWriter) GetStatusString() string {
	return fmt.Sprintf("%d", srw.status)
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
