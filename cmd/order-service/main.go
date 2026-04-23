package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"observability-demo/pkg/telemetry"
)

// Prometheus metrics
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "order_http_requests_total",
			Help: "Total number of HTTP requests to order-service",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "order_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds for order-service",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	ordersCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "orders_created_total",
			Help: "Total number of orders created",
		},
	)
)

// Domain types
type Order struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Status    string    `json:"status"`
	Product   *Product  `json:"product,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description,omitempty"`
	InStock     bool    `json:"in_stock"`
}

type CreateOrderRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// In-memory order store
var (
	orders   = make(map[string]*Order)
	ordersMu sync.RWMutex
	orderSeq int
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize OpenTelemetry tracer
	tp, err := telemetry.InitTracer(ctx, "order-service")
	if err != nil {
		log.Fatalf("failed to init tracer: %v", err)
	}
	defer tp.Shutdown(ctx)

	// Initialize structured logger
	logger, err := telemetry.InitLogger("order-service")
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	// Register Prometheus metrics
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration, ordersCreatedTotal)

	// Product service URL from environment
	productServiceURL := os.Getenv("PRODUCT_SERVICE_URL")
	if productServiceURL == "" {
		productServiceURL = "http://localhost:8082"
	}

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth(logger))
	mux.HandleFunc("POST /orders", handleCreateOrder(logger, productServiceURL))
	mux.HandleFunc("GET /orders", handleListOrders(logger))
	mux.Handle("GET /metrics", promhttp.Handler())

	// Middleware chain: OTel tracing → Prometheus metrics + logging → handlers
	var handler http.Handler = mux
	handler = telemetry.InstrumentHandler(handler, logger, httpRequestsTotal, httpRequestDuration)
	handler = otelhttp.NewHandler(handler, "order-service")

	server := &http.Server{
		Addr:    ":8081",
		Handler: handler,
	}

	// Start server
	go func() {
		logger.Info("order-service starting", zap.String("addr", ":8081"))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("shutting down order-service...")
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)
}

func handleHealth(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "order-service",
		})
	}
}

func handleCreateOrder(logger *zap.Logger, productServiceURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		span := trace.SpanFromContext(ctx)
		traceID := span.SpanContext().TraceID().String()

		// Parse request body
		var req CreateOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("invalid request body",
				zap.Error(err),
				zap.String("trace_id", traceID),
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}

		// Validate input
		if req.ProductID == "" || req.Quantity <= 0 {
			logger.Warn("invalid order parameters",
				zap.String("product_id", req.ProductID),
				zap.Int("quantity", req.Quantity),
				zap.String("trace_id", traceID),
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "product_id and positive quantity required"})
			return
		}

		logger.Info("creating order - fetching product info",
			zap.String("product_id", req.ProductID),
			zap.String("trace_id", traceID),
		)

		// Call product-service to validate and get product details
		product, err := getProduct(ctx, logger, productServiceURL, req.ProductID)
		if err != nil {
			logger.Error("failed to get product from product-service",
				zap.Error(err),
				zap.String("product_id", req.ProductID),
				zap.String("trace_id", traceID),
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to get product: %s", err.Error())})
			return
		}

		// Create order
		ordersMu.Lock()
		orderSeq++
		order := &Order{
			ID:        fmt.Sprintf("ORD-%04d", orderSeq),
			ProductID: req.ProductID,
			Quantity:  req.Quantity,
			Status:    "created",
			Product:   product,
			CreatedAt: time.Now(),
		}
		orders[order.ID] = order
		ordersMu.Unlock()

		// Update span with order details
		ordersCreatedTotal.Inc()
		span.SetAttributes(
			attribute.String("order.id", order.ID),
			attribute.String("order.product_id", order.ProductID),
			attribute.Int("order.quantity", order.Quantity),
		)

		logger.Info("order created successfully",
			zap.String("order_id", order.ID),
			zap.String("product_id", order.ProductID),
			zap.String("product_name", product.Name),
			zap.Float64("product_price", product.Price),
			zap.Int("quantity", order.Quantity),
			zap.String("trace_id", traceID),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(order)
	}
}

func handleListOrders(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		traceID := span.SpanContext().TraceID().String()

		ordersMu.RLock()
		result := make([]*Order, 0, len(orders))
		for _, o := range orders {
			result = append(result, o)
		}
		ordersMu.RUnlock()

		logger.Info("listing orders",
			zap.Int("count", len(result)),
			zap.String("trace_id", traceID),
		)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// getProduct calls product-service with trace context propagation
func getProduct(ctx context.Context, logger *zap.Logger, baseURL, productID string) (*Product, error) {
	tracer := otel.Tracer("order-service")
	ctx, span := tracer.Start(ctx, "get-product",
		trace.WithAttributes(attribute.String("product.id", productID)),
	)
	defer span.End()

	url := fmt.Sprintf("%s/products/%s", baseURL, productID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	// Inject trace context into outgoing HTTP request headers
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("request to product-service failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("product-service returned status %d: %s", resp.StatusCode, string(body))
		span.RecordError(err)
		return nil, err
	}

	var product Product
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to decode product response: %w", err)
	}

	span.SetAttributes(
		attribute.String("product.name", product.Name),
		attribute.Float64("product.price", product.Price),
	)

	logger.Info("product fetched from product-service",
		zap.String("product_id", product.ID),
		zap.String("product_name", product.Name),
		zap.String("trace_id", span.SpanContext().TraceID().String()),
	)

	return &product, nil
}
