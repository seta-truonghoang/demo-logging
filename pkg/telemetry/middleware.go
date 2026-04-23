package telemetry

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// GinMetricsMiddleware returns a Gin middleware that records Prometheus metrics
// and structured request logs with trace ID correlation.
// It skips /metrics and /health endpoints to avoid noise.
func GinMetricsMiddleware(
	logger *zap.Logger,
	requestsTotal *prometheus.CounterVec,
	requestDuration *prometheus.HistogramVec,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip instrumenting the metrics and health endpoints
		if c.Request.URL.Path == "/metrics" || c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		start := time.Now()

		// Process request
		c.Next()

		duration := time.Since(start).Seconds()
		status := fmt.Sprintf("%d", c.Writer.Status())

		// Use FullPath() for registered route pattern (lower cardinality)
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// Record Prometheus metrics
		requestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		requestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)

		// Extract trace ID from span context for log correlation
		span := trace.SpanFromContext(c.Request.Context())
		traceID := span.SpanContext().TraceID().String()

		logger.Info("request completed",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Float64("duration_seconds", duration),
			zap.String("trace_id", traceID),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}
