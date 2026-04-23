# 🔭 Observability Demo

Hệ thống demo đầy đủ 3 trụ cột Observability: **Logging**, **Metrics**, **Tracing** với Grafana visualization.

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

### 1. Khởi động toàn bộ stack

```bash
docker compose up -d --build
```

### 2. Kiểm tra services

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

### 3. Generate traffic cho demo

```bash
chmod +x scripts/generate-traffic.sh
./scripts/generate-traffic.sh 20
```

### 4. Mở Grafana

- URL: [http://localhost:3000](http://localhost:3000)
- Username: `admin`
- Password: `admin`

## 📊 Xem data trong Grafana

### Dashboard
Truy cập dashboard đã được provision sẵn:
- **Observability Demo - Services Overview**: [http://localhost:3000/d/services-overview](http://localhost:3000/d/services-overview)

### Explore - Logs (Loki)
1. Vào **Explore** → chọn datasource **Loki**
2. Query: `{service="order-service"}`
3. Thấy structured JSON logs với `trace_id` → click vào `TraceID` để jump sang Tempo

### Explore - Metrics (Prometheus)
1. Vào **Explore** → chọn datasource **Prometheus**
2. Một số queries hữu ích:
   - `rate(order_http_requests_total[1m])` - Request rate
   - `histogram_quantile(0.95, rate(order_http_request_duration_seconds_bucket[5m]))` - P95 latency
   - `orders_created_total` - Total orders

### Explore - Traces (Tempo)
1. Vào **Explore** → chọn datasource **Tempo**
2. Search traces → thấy distributed traces qua cả 2 services
3. Click vào trace → thấy waterfall view với spans từ order-service → product-service

## 🔄 Correlation Flow

```
Log (Loki) → trace_id → Trace (Tempo) → service name → Metrics (Prometheus)
```

- **Log → Trace**: Click vào `TraceID` derived field trong Loki logs
- **Trace → Log**: Từ trace detail, click "Logs for this span" để xem logs liên quan
- **Trace → Metrics**: Từ trace detail, xem metrics liên quan

## 📁 Project Structure

```
demo/
├── cmd/
│   ├── order-service/main.go     # Order API + calls product-service
│   └── product-service/main.go   # Product catalog API
├── pkg/
│   └── telemetry/
│       ├── telemetry.go          # OTel tracer + Zap logger init
│       └── middleware.go         # Prometheus metrics + logging middleware
├── configs/
│   ├── prometheus.yml            # Scrape config
│   ├── loki.yml                  # Loki storage config
│   ├── promtail.yml              # Docker log collection
│   ├── tempo.yml                 # Trace storage config
│   ├── otel-collector.yml        # OTLP receiver → Tempo exporter
│   └── grafana/
│       ├── datasources.yml       # Auto-provision datasources
│       ├── dashboards.yml        # Dashboard provisioning
│       └── dashboards/
│           └── overview.json     # Services overview dashboard
├── scripts/
│   └── generate-traffic.sh       # Traffic generator for demo
├── docker-compose.yml            # All 8 services
├── Dockerfile                    # Multi-stage Go build
├── go.mod
└── README.md
```

## 🛑 Dừng stack

```bash
docker compose down
```

Xoá cả data:
```bash
docker compose down -v
```
