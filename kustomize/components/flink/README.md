# Real-time Event Analytics with Apache Flink

This component deploys an [Apache Flink](https://flink.apache.org/) cluster and execution runner to perform real-time
stream processing and windowed analytics over microservice events in Kafka. It includes a JobManager master node,
TaskManager worker nodes, and an automated SQL Runner Job to submit continuous Flink SQL queries.

### Check Running Jobs:
Port `:30181` to view active streaming jobs in the Flink Web UI

## Quick Start

From the `kustomize/` folder at the root level of this repository, execute this command:

```bash
kustomize edit add component components/flink
```

This will update the `kustomize/kustomization.yaml` file which could be similar to:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/flink
```

You can locally render these manifests by running `kubectl kustomize .` as well as deploying them by running `kubectl apply -k .`.

## Components

### JobManager
The master node responsible for resource management, job scheduling, checkpoint coordination, and serving the management Web UI.

**Deployment Details:**
- **Ports:** 8081 (Web UI & REST API), 6123 (RPC), 6124 (BLOB Server)
- **Access:** :`:30181` which bind to 8081

**Configuration:**
- Manages execution graphs and job submissions from the SQL Client
- Exposes metrics and running job status via Web UI
- Configured via `flink-config` ConfigMap (`FLINK_PROPERTIES`)

### TaskManager
The worker process that executes stream processing tasks, maintains state, and outputs sink results.

**Deployment Details:**
- **Replicas:** 1 (scalable)
- **Slots:** 4 Task Slots per worker
- **Memory:** 2600m Process Memory

### Flink SQL Runner (Job)
A Kubernetes batch Job that initializes connector dependencies and submits long-running Flink SQL scripts to the JobManager.

**Deployment Details:**
- **Image:** `flink:1.19.0`
- **Execution:** Runs `sql-client.sh` in embedded mode to submit streaming queries

**Configuration:**
- Uses an init container (`wait-and-setup`) to fetch `flink-connector-jdbc`, `postgresql` and `flink-sql-connector-kafka` JAR from Maven Central and wait for the JobManager REST API
- Passes configuration environment variables via `/docker-entrypoint.sh`
- Combines all sql scripts inside the `sql` folder into a single  `combined_jobs.sql` file for execution

## SQL Analytics Scripts

### `00_setup.sql`
Defines the data sources for processing product events:
- **Kafka Source (`product_events`):** Reads JSON-formatted events from the `product-events` Kafka topic with ISO-8601 timestamps and watermark strategy (`event_time - INTERVAL '5' SECOND`).

### `01_raw_orders.sql`
- **Postgres Sink (`raw_order_events_sink`):** Outputs raw `ORDER` events to the `raw_order_events` postgres table for persistent storage.
- **Continuous Query:** Executes a continuous query to store the raw order events.

### `10_product_sales_1m.sql`
- **Postgres Sink (`product_sales_1m_sink`):** Outputs aggregated results to `product_sales_1m` postgres table.
- **Continuous Query:** Executes a 1-minute tumbling window aggregation on `ORDER` events, computing total units bought per product SKU.

### `11_product_clicks_1m.sql`
- **Postgres Sink (`product_clicks_1m_sink`):** Outputs aggregated results to `product_clicks_1m` postgres table.
- **Continuous Query:** Executes a 1-minute tumbling window aggregation on `VIEW` events, computing total clicks/views per product SKU.

### `20_view_to_cart_1h.sql`
- **Postgres Sink (`product_view_to_cart_1h_sink`):** Outputs aggregated results to `product_view_to_cart_1h` postgres table.
- **Continuous Query:** Executes a 1-hour tumbling window aggregation on `VIEW` and `CART` events, computing the number of views that lead to cart additions.

### `21_average_order_value_1h.sql`
- **Postgres Sink (`order_aov_1h_sink`):** Outputs aggregated results to `order_aov_1h` postgres table.
- **Continuous Query:** Executes a 1-hour tumbling window aggregation on `ORDER` events, computing the average order value.
