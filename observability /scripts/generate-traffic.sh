#!/bin/bash
# generate-traffic.sh - Generate sample traffic for the observability demo
# This script creates orders with random products to populate metrics, logs, and traces.

set -e

ORDER_SERVICE_URL="${ORDER_SERVICE_URL:-http://localhost:8081}"
PRODUCT_SERVICE_URL="${PRODUCT_SERVICE_URL:-http://localhost:8082}"
ITERATIONS="${1:-20}"

echo "╔══════════════════════════════════════════════╗"
echo "║   Observability Demo - Traffic Generator     ║"
echo "╠══════════════════════════════════════════════╣"
echo "║  Order Service:   $ORDER_SERVICE_URL          "
echo "║  Product Service: $PRODUCT_SERVICE_URL        "
echo "║  Iterations:      $ITERATIONS                 "
echo "╚══════════════════════════════════════════════╝"
echo ""

# Wait for services to be ready
echo "⏳ Waiting for services to be ready..."
for i in {1..30}; do
    if curl -sf "$ORDER_SERVICE_URL/health" > /dev/null 2>&1 && \
       curl -sf "$PRODUCT_SERVICE_URL/health" > /dev/null 2>&1; then
        echo "✅ Services are ready!"
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "❌ Services not ready after 30 seconds. Exiting."
        exit 1
    fi
    sleep 1
done

echo ""
echo "📦 Listing available products..."
curl -s "$PRODUCT_SERVICE_URL/products" | python3 -m json.tool 2>/dev/null || \
    curl -s "$PRODUCT_SERVICE_URL/products"
echo ""

echo "🚀 Generating $ITERATIONS orders..."
echo "─────────────────────────────────────────────"

for i in $(seq 1 "$ITERATIONS"); do
    # Random product ID (1-5) and quantity (1-5)
    product_id=$((RANDOM % 5 + 1))
    quantity=$((RANDOM % 5 + 1))

    echo -n "[$i/$ITERATIONS] Creating order: product=$product_id, qty=$quantity ... "

    response=$(curl -s -w "\n%{http_code}" -X POST "$ORDER_SERVICE_URL/orders" \
        -H "Content-Type: application/json" \
        -d "{\"product_id\": \"$product_id\", \"quantity\": $quantity}")

    http_code=$(echo "$response" | tail -1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" = "201" ]; then
        order_id=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "?")
        echo "✅ $order_id (HTTP $http_code)"
    else
        echo "❌ Failed (HTTP $http_code)"
    fi

    # Random delay between requests (0.2-1 seconds)
    sleep_time=$(echo "scale=1; ($RANDOM % 15 + 5) / 10" | bc)
    sleep "$sleep_time"
done

echo ""
echo "─────────────────────────────────────────────"
echo "📋 Listing all orders..."
curl -s "$ORDER_SERVICE_URL/orders" | python3 -m json.tool 2>/dev/null || \
    curl -s "$ORDER_SERVICE_URL/orders"

echo ""
echo "✨ Traffic generation complete!"
echo ""
echo "📊 View results in Grafana: http://localhost:3000"
echo "   Username: admin"
echo "   Password: admin"
echo ""
echo "🔍 Quick links:"
echo "   Dashboard:  http://localhost:3000/d/services-overview"
echo "   Explore Logs:    http://localhost:3000/explore?orgId=1&left={\"datasource\":\"loki\"}"
echo "   Explore Metrics: http://localhost:3000/explore?orgId=1&left={\"datasource\":\"prometheus\"}"
echo "   Explore Traces:  http://localhost:3000/explore?orgId=1&left={\"datasource\":\"tempo\"}"
