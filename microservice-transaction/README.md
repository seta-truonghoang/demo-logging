# Microservice Transaction Patterns Demos

Minimal, self-contained demos for several distributed-transaction patterns (Monolithic, Saga, 2PC) implemented in Go with Gin and GORM.

## Prerequisites

- Docker and Docker Compose installed

## How to Run

We use Docker Compose **profiles** to run different demo scenarios independently without conflicting with each other. The core infrastructure (PostgreSQL, Kafka, Zookeeper, Kafdrop) runs by default in all profiles.

First, build the images:
```bash
docker-compose build
```

### Scenario 1: Monolithic Transaction

This runs a single monolithic Go app that uses standard PostgreSQL database transactions to transfer funds atomically.

```bash
docker-compose --profile monolithic up -d
```
- **Service Port**: `8080`
- **Test Command**:
  ```bash
  curl -X POST http://localhost:8080/transfer \
  -H 'Content-Type: application/json' \
  -d '{"from": "alice", "to": "bob", "amount": 100}'
  ```

### Scenario 2: Microservice Saga (Choreography)

This runs two microservices (`order-service` and `payment-service`) that communicate asynchronously via Kafka events.

```bash
docker-compose --profile saga up -d
```
- **Order Service Port**: `8081`
- **Payment Service Port**: `8082`
- **Test Command**:
  ```bash
  curl -X POST http://localhost:8081/orders \
  -H 'Content-Type: application/json' \
  -d '{"account_id": "alice", "amount": 100}'
  ```
*(Check the logs of both services to see the events propagating and the order status updating to CONFIRMED. Send an amount > 1000 to see it rollback/cancel).*

### Scenario 3: Microservice Two-Phase Commit (2PC)

This runs three microservices (`coordinator`, `order-service`, and `payment-service`) that communicate synchronously via HTTP endpoints (`/prepare`, `/commit`, `/rollback`).

```bash
docker-compose --profile 2pc up -d
```
- **Coordinator Port**: `8085`
- **Order Service Port**: `8083`
- **Payment Service Port**: `8084`
- **Test Command**:
  ```bash
  curl -X POST http://localhost:8085/checkout \
  -H 'Content-Type: application/json' \
  -d '{"account_id": "bob", "amount": 100}'
  ```
*(Check the coordinator logs to observe Phase 1 (Prepare) and Phase 2 (Commit). Send an amount > 500 to see the Prepare phase fail and trigger a Rollback).*

---

### Cleaning Up

To stop the services and remove containers (including the shared infrastructure):

```bash
docker-compose down
```

To view the database directly, you can connect to the PostgreSQL instance running on `localhost:5432` with user `demo` and password `demo`. Kafka messages can be viewed via Kafdrop at `http://localhost:9000`.
