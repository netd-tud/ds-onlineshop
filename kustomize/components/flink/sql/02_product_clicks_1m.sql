-- Aggregate product view over one minute
CREATE TABLE IF NOT EXISTS product_clicks_1m_sink (
  sku STRING,
  total_clicks BIGINT,
  window_start TIMESTAMP(3),
  PRIMARY KEY (sku, window_start) NOT ENFORCED
) WITH (
  'connector' = 'jdbc',
  'url' = 'jdbc:postgresql://postgres:5432/analytics_db',
  'table-name' = 'product_clicks_1m',
  'username' = 'user',
  'password' = 'password',
  'sink.buffer-flush.max-rows' = '100',
  'sink.buffer-flush.interval' = '1s'
);

INSERT INTO product_clicks_1m_sink
SELECT
  sku,
  COUNT(*) AS total_clicks,
  TUMBLE_START(event_time, INTERVAL '1' MINUTE) AS window_start
FROM product_events
WHERE event_type = 'VIEW'
GROUP BY
  sku,
  TUMBLE(event_time, INTERVAL '1' MINUTE);
