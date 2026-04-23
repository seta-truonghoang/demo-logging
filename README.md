# 🔭 Observability Demo

A comprehensive Observability demo system featuring the three pillars: **Logging**, **Metrics**, and **Tracing** with unified visualization in Grafana.

## 🏗 Architecture

```
┌─────────────────┐     HTTP      ┌──────────────────┐
│  order-service   │ ──────────► │  product-service   │
│     :8081        │              │      :8082         │
└────────┬─────────┘              └────────┬───────────┘
         │                                  │
         │ OTLP gRPC                       │ OTLP gRPC
         │                                  │
         ▼                                  ▼
┌──────────────────┐    OTLP     ┌──────────────────┐
│  OTel Collector  │ ──────────► │      Tempo        │
│     :4317        │             │     :3200         │
└──────────────────┘             └──────────────────┘

┌──────────────────┐   push     ┌──────────────────┐
│    Promtail      │ ─────────► │       Loki        │
│  (Docker logs)   │            │      :3100        │
└──────────────────┘            └──────────────────┘

┌──────────────────┐   scrape   ┌──────────────────┐
│   Prometheus     │ ─────────► │  Go Services      │
│     :9090        │            │  /metrics          │
└──────────────────┘            └──────────────────┘

         ┌─────────────────────────────────┐
         │           Grafana :3000          │
         │  ┌─────────┬────────┬────────┐  │
         │  │  Loki   │ Prom   │ Tempo  │  │
         │  │ (logs)  │(metric)│(traces)│  │
         │  └─────────┴────────┴────────┘  │
         └─────────────────────────────────┘
```

## 🚀 Quick Start

### 1. Start the entire stack

```bash
docker compose up -d --build
```

### 2. Verify Services

```bash
# Health checks
curl http://localhost:8081/health   # order-service
curl http://localhost:8082/health   # product-service

# List products
curl http://localhost:8082/products

# Create an order
curl -X POST http://localhost:8081/orders \
  -H "Content-Type: application/json" \
  -d '{"product_id": "1", "quantity": 2}'

# List orders
curl http://localhost:8081/orders
```

### 3. Generate Traffic for Demo

```bash
chmod +x scripts/generate-traffic.sh
./scripts/generate-traffic.sh 20
```

### 4. Access Grafana

- URL: [http://localhost:3000](http://localhost:3000)
- Username: `admin`
- Password: `admin`

## 📊 Viewing Data in Grafana

### Dashboard
Access the pre-provisioned dashboard:
- **Observability Demo - Services Overview**: [http://localhost:3000/d/services-overview](http://localhost:3000/d/services-overview)

### Explore - Logs (Loki)
1. Go to **Explore** → select **Loki** datasource.
2. Query: `{service="order-service"}`
3. You will see structured JSON logs with a `trace_id` → click on the `TraceID` label to jump directly to Tempo.

### Explore - Metrics (Prometheus)
1. Go to **Explore** → select **Prometheus** datasource.
2. Useful queries:
   - `rate(order_http_requests_total[1m])` - Request rate
   - `histogram_quantile(0.95, rate(order_http_request_duration_seconds_bucket[5m]))` - P95 latency
   - `orders_created_total` - Total orders counter

### Explore - Traces (Tempo)
1. Go to **Explore** → select **Tempo** datasource.
2. Search for traces → view distributed traces spanning across both services.
3. Click on a trace → waterfall view showing spans from order-service to product-service.

## 🔄 Correlation Flow

```
Log (Loki) → trace_id → Trace (Tempo) → service name → Metrics (Prometheus)
```

- **Log → Trace**: Click on the `TraceID` derived field in Loki logs.
- **Trace → Log**: From the trace detail view, click "Logs for this span" to see related entries.
- **Trace → Metrics**: From the trace detail view, view related metrics for the service.

## 📁 Project Structure

```
demo/
├── cmd/
│   ├── order-service/main.go     # Order API calling product-service
│   └── product-service/main.go   # Product catalog API
├── pkg/
│   └── telemetry/
│       ├── telemetry.go          # OTel tracer & Zap logger initialization
│       └── middleware.go         # Prometheus metrics & logging middleware
├── configs/
│   ├── prometheus.yml            # Scrape configuration
│   ├── loki.yml                  # Loki storage configuration
│   ├── promtail.yml              # Docker log collection configuration
│   ├── tempo.yml                 # Trace storage configuration
│   ├── otel-collector.yml        # OTLP receiver to Tempo exporter
│   └── grafana/
│       ├── datasources.yml       # Auto-provisioned datasources
│       ├── dashboards.yml        # Dashboard provisioning configuration
│       └── dashboards/
│           └── overview.json     # Services overview dashboard definition
├── scripts/
│   └── generate-traffic.sh       # Traffic generator script
├── docker-compose.yml            # Full stack orchestration (8 services)
├── Dockerfile                    # Multi-stage Go build (Go 1.25.5)
├── go.mod
└── README.md
```

## 🛑 Stop the Stack

```bash
docker compose down
```

To remove all data volumes as well:
```bash
docker compose down -v
```
