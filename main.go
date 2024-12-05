package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ResponseWriter is a custom wrapper around http.ResponseWriter that captures the status code.
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// NewResponseWriter creates a new instance of responseWriter with a default status code of 200.
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{w, http.StatusOK}
}

// WriteHeader overrides the default WriteHeader to capture the status code.
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Define Prometheus metrics using promauto to avoid manual registration.
var (
	// totalRequests tracks the total number of HTTP requests.
	totalRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests received.",
		},
		[]string{"path"},
	)

	// responseStatus tracks the status codes of HTTP responses.
	responseStatus = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "response_status",
			Help: "Counts HTTP response statuses.",
		},
		[]string{"status"},
	)

	// httpDuration tracks the duration of HTTP requests.
	httpDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_response_time_seconds",
			Help: "Duration of HTTP requests in seconds.",
		},
		[]string{"path"},
	)
)

// prometheusMiddleware is a middleware that captures and records Prometheus metrics for each HTTP request.
func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Retrieve the path template for the current route.
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()

		// Start the timer for request duration.
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))
		defer timer.ObserveDuration() // Record the duration after the request is handled.

		// Use custom response writer to capture the status code.
		rw := NewResponseWriter(w)
		next.ServeHTTP(rw, r)

		// Record metrics for the response.
		statusCode := rw.statusCode
		responseStatus.WithLabelValues(strconv.Itoa(statusCode)).Inc()
		totalRequests.WithLabelValues(path).Inc()
	})
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	// Respond with a 200 OK status and a simple JSON message.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := fmt.Fprintf(w, `{"status": "healthy"}`)
	if err != nil {
		return
	}
}

func main() {
	// Create a new Gorilla Mux router.
	router := mux.NewRouter()

	// Apply the Prometheus middleware to all routes.
	router.Use(prometheusMiddleware)

	// Serve static files from the "static" directory under the "/static/" URL path.
	router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))),
	)

	// Serve the root URL ("/") to render the index.html or any other page in your static folder.
	router.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./static/"))))

	// Expose Prometheus metrics at the "/prometheus" endpoint.
	router.Handle("/prometheus", promhttp.Handler())

	// Define a health check endpoint at "/health".
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// Start the HTTP server on port 9000.
	fmt.Println("Serving requests on port 9000")
	if err := http.ListenAndServe(":9000", router); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
