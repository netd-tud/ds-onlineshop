-- Aggregate product sales over one minute
CREATE TABLE IF NOT EXISTS product_sales_1m_sink (
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

INSERT INTO product_sales_1m_sink
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
