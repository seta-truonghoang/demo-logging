package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"observability-demo/pkg/telemetry"
)

// Prometheus metrics
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "product_http_requests_total",
			Help: "Total number of HTTP requests to product-service",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "product_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds for product-service",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

// Product represents a product in the catalog
type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	InStock     bool    `json:"in_stock"`
}

// Hardcoded product catalog
var products = map[string]*Product{
	"1": {ID: "1", Name: "Laptop Pro 16", Price: 1299.99, Description: "High-performance laptop with 16-inch Retina display", InStock: true},
	"2": {ID: "2", Name: "Wireless Mouse", Price: 29.99, Description: "Ergonomic wireless mouse with USB-C charging", InStock: true},
	"3": {ID: "3", Name: "Mechanical Keyboard", Price: 89.99, Description: "RGB mechanical keyboard with Cherry MX switches", InStock: true},
	"4": {ID: "4", Name: "4K Monitor", Price: 449.99, Description: "27-inch 4K IPS monitor with HDR support", InStock: false},
	"5": {ID: "5", Name: "USB-C Hub", Price: 49.99, Description: "7-in-1 USB-C hub with HDMI, USB-A, SD card reader", InStock: true},
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize OpenTelemetry tracer
	tp, err := telemetry.InitTracer(ctx, "product-service")
	if err != nil {
		log.Fatalf("failed to init tracer: %v", err)
	}
	defer tp.Shutdown(ctx)

	// Initialize structured logger
	logger, err := telemetry.InitLogger("product-service")
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	// Register Prometheus metrics
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(otelgin.Middleware("product-service"))
	router.Use(telemetry.GinMetricsMiddleware(logger, httpRequestsTotal, httpRequestDuration))

	// Routes
	router.GET("/health", handleHealth)
	router.GET("/products/:id", handleGetProduct(logger))
	router.GET("/products", handleListProducts(logger))
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	server := &http.Server{
		Addr:    ":8082",
		Handler: router,
	}

	// Start server
	go func() {
		logger.Info("product-service starting", zap.String("addr", ":8082"))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("shutting down product-service...")
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)
}

func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "product-service",
	})
}

func handleGetProduct(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		span := trace.SpanFromContext(ctx)
		traceID := span.SpanContext().TraceID().String()

		// Extract product ID from URL path parameter
		id := c.Param("id")
		span.SetAttributes(attribute.String("product.id", id))

		logger.Info("fetching product",
			zap.String("product_id", id),
			zap.String("trace_id", traceID),
		)

		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)

		product, ok := products[id]
		if !ok {
			logger.Warn("product not found",
				zap.String("product_id", id),
				zap.String("trace_id", traceID),
			)
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}

		span.SetAttributes(
			attribute.String("product.name", product.Name),
			attribute.Float64("product.price", product.Price),
			attribute.Bool("product.in_stock", product.InStock),
		)

		logger.Info("product found",
			zap.String("product_id", product.ID),
			zap.String("product_name", product.Name),
			zap.Float64("product_price", product.Price),
			zap.Bool("in_stock", product.InStock),
			zap.String("trace_id", traceID),
		)

		c.JSON(http.StatusOK, product)
	}
}

func handleListProducts(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		traceID := span.SpanContext().TraceID().String()

		logger.Info("listing all products",
			zap.Int("total_products", len(products)),
			zap.String("trace_id", traceID),
		)

		result := make([]*Product, 0, len(products))
		for _, p := range products {
			result = append(result, p)
		}

		c.JSON(http.StatusOK, result)
	}
}
