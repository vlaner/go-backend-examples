package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var requestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "backend",
		Name:      "http_request_duration_seconds",
		Help:      "Request duration in seconds",
		Buckets:   []float64{0.1, 0.15, 0.2, 0.25, 0.3, 0.5, 1, 5},
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
	randomDuration := time.Duration(rand.Intn(1000)+100) * time.Millisecond
	time.Sleep(randomDuration)
	w.Write([]byte("hello"))
}

func main() {
	prometheus.MustRegister(requestDuration, requestDurationSummary)

	appMux := http.NewServeMux()
	appMux.HandleFunc("/", MetricsMiddleware(Index))
	appMux.Handle("/metrics", promhttp.Handler())

	go func() {
		log.Fatal(http.ListenAndServe(":8081", appMux))
	}()

	select {}
}
