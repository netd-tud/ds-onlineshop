-- Persistent Order Sink
CREATE TABLE IF NOT EXISTS raw_order_events_sink (
  event_id STRING,
  event_time TIMESTAMP(3),
  event_type STRING,
  order_id STRING,
  session_id STRING,
  producer STRING,
  currency_code STRING,
  price_units BIGINT,
  price_nanos INT,
  amount DOUBLE,
  PRIMARY KEY (event_id) NOT ENFORCED
) WITH (
  'connector' = 'jdbc',
  'url' = 'jdbc:postgresql://postgres:5432/analytics_db',
  'table-name' = 'raw_order_events',
  'username' = 'user',
  'password' = 'password',
  'sink.buffer-flush.max-rows' = '1000',
  'sink.buffer-flush.interval' = '1s',
  'sink.max-retries' = '3'
);

INSERT INTO raw_order_events_sink
SELECT
  event_id,
  CAST(event_time AS TIMESTAMP(3)) AS event_time,
  event_type,
  order_id,
  session_id,
  producer,
  price.currency_code AS currency_code,
  price.units AS price_units,
  price.nanos AS price_nanos,
  price.units + (CAST(price.nanos AS DOUBLE) / 1000000000.0) AS amount
FROM order_events;
