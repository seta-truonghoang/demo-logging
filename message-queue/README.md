# ⚡ Kafka Event Streaming & Consumer Groups Demo

An interactive, visual, and educational demonstration system for **Apache Kafka** event streaming patterns and **Consumer Groups** in Go.

This demo features a single-binary Go backend that hosts a **stunning, neon-glowing Web UI dashboard**. The dashboard provides interactive message dispatching, real-time log streaming, and an automated SVG-based topology visualizer that maps events traveling from the Producer to specific Partitions, and from those Partitions to the assigned active Worker instances.

---

## 🏗 System Architecture

```
┌────────────────────────────────────────────────────────┐
│                      PRODUCER                          │
│  - Select strategy: Key Hash / Round Robin / Direct   │
│  - Dispatch custom JSON payloads (Presets: Order, etc.)│
└──────────────────────────┬─────────────────────────────┘
                           │ HTTP POST /api/produce
                           ▼
┌────────────────────────────────────────────────────────┐
│             KAFKA BROKER (KRaft mode :9094)            │
│  Topic: `ecommerce-events`                             │
│  ┌─────────────────┬─────────────────┬─────────────────┐
│  │   Partition 0   │   Partition 1   │   Partition 2   │
│  └────────┬────────┴────────┬────────┴────────┬────────┘
└───────────┼─────────────────┼─────────────────┼────────┘
            │                 │                 │
            │ Assigned        │ Assigned        │ Assigned
            ▼                 ▼                 ▼
┌────────────────────────────────────────────────────────┐
│     CONSUMER GROUP: `order-processing-group`           │
│  - Worker 1 (Active/Inactive toggle)                   │
│  - Worker 2 (Active/Inactive toggle)                   │
│  - Worker 3 (Active/Inactive toggle)                   │
└────────────────────────────────────────────────────────┘
```

---

## ✨ Features

- **Apache Kafka in KRaft Mode (No ZooKeeper)**: Uses the modern metadata quorum controller mode for near-instant cluster startups and reduced resource footprint.
- **Interactive SVG Topology Mapping**: Draws real-time horizontal Bezier curves linking active workers to the specific topic partitions assigned to them by Kafka.
- **Sparkle/Packet Animation**: When a message is produced, a glowing packet animates along the connection curve to its target partition, and from there to the processing worker.
- **Live Consumer Group Rebalancing**: Stop or start any of the 3 worker instances on-the-fly via simple toggle switches. Watch how Kafka dynamically triggers a group rebalance and reshuffles partition assignments across the remaining active consumers.
- **Key-Based Hash Partitioning**: Demonstrates how Kafka hashes the message key (e.g., `order_id`) to direct related events to the *same partition*, ensuring strict FIFO ordering per entity.
- **Custom Processing Speed**: Adjust the simulated network/processing delay of workers in real-time to witness message queues stack up and resolve under pressure.
- **Real-Time Log Stream (SSE)**: Built-in Server-Sent Events broker that pumps colored, filterable logs directly from Go routines (Producers, Consumers, Cluster Events) to the UI console.

---

## 📁 Project Structure

```
message-queue/
├── main.go               # Consolidated Go server (Admin client, Producer, Consumers, Logs SSE, API)
├── Dockerfile            # Multi-stage statically-linked Go Alpine builder
├── docker-compose.yml    # Kafka KRaft container & Go web server orchestration
├── go.mod                # Go module manifest (using segmentio/kafka-go)
├── go.sum                # Package checksum hashes
└── web/
    └── index.html        # Embedded HTML dashboard (translucent styling, SVG connectors, dynamic state)
```

---

## 🚀 Quick Start (Recommended)

### 1. Build and Start the Entire Stack
Run Docker Compose in the project root to build the Go app and pull down the Kafka broker.
```bash
docker compose up -d --build
```

### 2. Verify Container Health
Wait 5–10 seconds for the Kafka container to complete initialization. Check that both containers are running happily:
```bash
docker compose ps
```

### 3. Open the Console Dashboard
Head over to your browser and access the interactive control center:
- **Web UI URL**: [http://localhost:8080](http://localhost:8080)

---

## 💻 Local Go Developer Mode (Running without containerizing Go)

If you wish to make quick changes to the Go files or run Go natively on your host machine while keeping Kafka inside Docker:

### 1. Run Kafka Container Only
```bash
docker compose up -d kafka
```

### 2. Launch the Go App Locally
The app will connect to the `EXTERNAL` listener mapped on `localhost:9094`:
```bash
go run main.go
```
The console is now served locally on [http://localhost:8080](http://localhost:8080).

---

## ⚙️ Kafka Concepts Demonstrated Live

### 1. Key-Based Hash Partitioning
1. Select **Hash-Key Partitioning** in the Producer Hub.
2. Put `order-abc` in the Message Key and press **Dispatch Event**. Note which partition (e.g., `Partition #2`) receives the message.
3. Send it again. Notice that as long as the key is `order-abc`, it **always** targets the same partition. This is how Kafka guarantees message sequencing for individual orders!
4. Change the key to `order-xyz` and dispatch. See it automatically jump to a different partition.

### 2. Round-Robin Distribution
1. Select **Round-Robin Partitioning** in the Producer Hub (which clears the key requirement).
2. Press **Dispatch Event** multiple times.
3. Watch the visual packets cycle sequentially between `Partition #0`, `Partition #1`, and `Partition #2`, distributing the processing load perfectly across the broker.

### 3. Consumer Group Live Rebalancing
1. Look at the **Group Monitor** on the right. With all 3 workers active, Kafka assigns exactly one partition to each worker:
   - `Worker-1` -> `Partition #0`
   - `Worker-2` -> `Partition #1`
   - `Worker-3` -> `Partition #2`
2. **Deactivate Worker-3** using its toggle switch.
3. In the streaming logs, you will immediately see a group rebalance take place.
4. Observe the new active assignments: one of the remaining active workers (e.g., `Worker-1`) has taken over `Partition #2` and is now processing both partitions.
5. **Re-activate Worker-3**. Kafka rebalances again, gracefully revoking `Partition #2` from `Worker-1` and assigning it back to `Worker-3`.

---

## 🛑 Clean Up

Shut down all containers and clean up Docker-allocated networks and volumes:
```bash
docker compose down -v
```
