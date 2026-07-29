CREATE TABLE IF NOT EXISTS product_view_to_cart_1h (
  window_start TIMESTAMP(3),
  window_end TIMESTAMP(3),
  sku STRING,
  total_views BIGINT,
  total_atc BIGINT,
  view_to_cart_rate DOUBLE,
  PRIMARY KEY (sku, window_start) NOT ENFORCED
) WITH (
  'connector' = 'jdbc',
  'url' = 'jdbc:postgresql://postgres:5432/analytics_db',
  'table-name' = 'product_view_to_cart_1h',
  'username' = 'user',
  'password' = 'password',
  'sink.buffer-flush.max-rows' = '100',
  'sink.buffer-flush.interval' = '1s'
);

INSERT INTO product_view_to_cart_1h
SELECT
  window_start,
  window_end,
  sku,
  COUNT(CASE WHEN event_type = 'VIEW' THEN 1 END) AS total_views,
  COUNT(CASE WHEN event_type = 'ATC' THEN 1 END) AS total_atc,
  CAST(COUNT(CASE WHEN event_type = 'ATC' THEN 1 END) AS DOUBLE) /
    NULLIF(COUNT(CASE WHEN event_type = 'VIEW' THEN 1 END), 0) AS view_to_cart_rate
FROM TABLE(
  TUMBLE(
    DATA => TABLE product_events,
    TIMECOL => DESCRIPTOR(event_time),
    SIZE => INTERVAL '1' HOUR
  )
)
WHERE event_type IN ('VIEW', 'ATC')
GROUP BY window_start, window_end, sku;
