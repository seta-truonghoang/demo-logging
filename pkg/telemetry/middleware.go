package telemetry

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// statusResponseWriter wraps http.ResponseWriter to capture the status code.
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *statusResponseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.statusCode = http.StatusOK
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

// InstrumentHandler wraps an http.Handler with Prometheus metrics recording
// and structured request logging including trace IDs.
// It skips /metrics and /health endpoints to avoid noise.
func InstrumentHandler(
	next http.Handler,
	logger *zap.Logger,
	requestsTotal *prometheus.CounterVec,
	requestDuration *prometheus.HistogramVec,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip instrumenting the metrics endpoint itself
		if r.URL.Path == "/metrics" || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		sw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(sw, r)

		duration := time.Since(start).Seconds()
		status := fmt.Sprintf("%d", sw.statusCode)
		path := r.URL.Path

		// Record Prometheus metrics
		requestsTotal.WithLabelValues(r.Method, path, status).Inc()
		requestDuration.WithLabelValues(r.Method, path).Observe(duration)

		// Extract trace ID from span context for log correlation
		span := trace.SpanFromContext(r.Context())
		traceID := span.SpanContext().TraceID().String()

		logger.Info("request completed",
			zap.String("method", r.Method),
			zap.String("path", path),
			zap.Int("status", sw.statusCode),
			zap.Float64("duration_seconds", duration),
			zap.String("trace_id", traceID),
			zap.String("remote_addr", r.RemoteAddr),
		)
	})
}
