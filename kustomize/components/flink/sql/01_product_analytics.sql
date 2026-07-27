-- Register the Kafka Source Table
CREATE TABLE IF NOT EXISTS product_events (
  event_id STRING,
  event_time TIMESTAMP_LTZ(3),
  event_type STRING,
  sku STRING,
  qty INT,
  price ROW<currency_code STRING, units BIGINT, nanos INT>,
  order_id STRING,
  session_id STRING,
  producer STRING,
  WATERMARK FOR event_time AS event_time - INTERVAL '5' SECOND
) WITH (
  'connector' = 'kafka',
  'topic' = 'product-events',
  'properties.bootstrap.servers' = 'analytics-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092',
  'properties.group.id' = 'flink-sql-runner',
  'scan.startup.mode' = 'earliest-offset',
  'scan.watermark.idle-timeout' = '5 s',
  'format' = 'json',
  'json.fail-on-missing-field' = 'false',
  'json.ignore-parse-errors' = 'true',
  'json.timestamp-format.standard' = 'ISO-8601'
);

CREATE TABLE IF NOT EXISTS product_events_aggregated_db (
  sku STRING,
  total_units_bought INT,
  window_start TIMESTAMP(3),
  PRIMARY KEY (sku, window_start) NOT ENFORCED
) WITH (
  'connector' = 'jdbc',
  'url' = 'jdbc:postgresql://postgres:5432/analytics_db',
  'table-name' = 'product_sales_1m',
  'username' = 'user',
  'password' = 'password',
  'sink.buffer-flush.max-rows' = '100',
  'sink.buffer-flush.interval' = '1s'
);

-- Continuous Aggregation Job
INSERT INTO product_events_aggregated_db
SELECT
  sku,
  SUM(qty) AS total_units_bought,
  TUMBLE_START(event_time, INTERVAL '1' MINUTE) AS window_start
FROM product_events
WHERE event_type = 'ORDER'
  AND qty IS NOT NULL
GROUP BY
  sku,
  TUMBLE(event_time, INTERVAL '1' MINUTE);
