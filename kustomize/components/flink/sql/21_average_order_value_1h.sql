CREATE TABLE IF NOT EXISTS order_aov_1h_sink (
  window_start TIMESTAMP(3),
  window_end TIMESTAMP(3),
  currency_code STRING,
  total_orders BIGINT,
  total_revenue DOUBLE,
  average_order_value DOUBLE,
  PRIMARY KEY (currency_code, window_start) NOT ENFORCED
) WITH (
  'connector' = 'jdbc',
  'url' = 'jdbc:postgresql://postgres:5432/analytics_db',
  'table-name' = 'order_aov_1h',
  'username' = 'user',
  'password' = 'password',
  'sink.buffer-flush.max-rows' = '100',
  'sink.buffer-flush.interval' = '1s'
);

INSERT INTO order_aov_1h
SELECT
  window_start,
  window_end,
  price.currency_code,
  COUNT(order_id) AS total_orders,
  SUM(price.units + (CAST(price.nanos AS DOUBLE) / 1000000000.0)) AS total_revenue,
  AVG(price.units + (CAST(price.nanos AS DOUBLE) / 1000000000.0)) AS average_order_value
FROM TABLE(
  TUMBLE(
    DATA => TABLE order_events,
    TIMECOL => DESCRIPTOR(event_time),
    SIZE => INTERVAL '1' HOUR
  )
)
WHERE event_type = 'COMPLETE'
GROUP BY
  window_start,
  window_end,
  price.currency_code;
