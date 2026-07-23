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

-- Debug: Print the incoming order events
CREATE TABLE IF NOT EXISTS product_events_aggregated (
  sku STRING,
  total_units_bought INT,
  ts TIMESTAMP(3)
) WITH (
  'connector' = 'print'
);
INSERT INTO product_events_aggregated
SELECT
  sku,
  qty AS total_units_bought,
  CAST(event_time AS TIMESTAMP(3)) AS ts
FROM product_events
WHERE event_type = 'ORDER';
