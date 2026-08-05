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
  window_start
FROM TABLE(
  TUMBLE(
    DATA => TABLE product_events,
    TIMECOL => DESCRIPTOR(event_time),
    SIZE => INTERVAL '1' MINUTE
  )
)
WHERE event_type = 'VIEW'
GROUP BY
  sku,
  window_start,
  window_end;
