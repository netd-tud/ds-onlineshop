CREATE TABLE IF NOT EXISTS product_recommendations_sink (
  sku_a STRING,
  sku_b STRING,
  co_purchase_count BIGINT,
  PRIMARY KEY (sku_a, sku_b) NOT ENFORCED
) WITH (
  'connector' = 'jdbc',
  'url' = 'jdbc:postgresql://postgres:5432/analytics_db',
  'table-name' = 'product_recommendations',
  'username' = 'user',
  'password' = 'password',
  'sink.buffer-flush.max-rows' = '100',
  'sink.buffer-flush.interval' = '1s'
);

SET 'execution.runtime-mode' = 'BATCH';

INSERT INTO product_recommendations_sink
SELECT a.sku AS sku_a, b.sku AS sku_b, COUNT(*) AS co_purchase_count
FROM product_events a
JOIN product_events b
  ON a.order_id = b.order_id
    AND a.sku < b.sku
    AND a.event_time BETWEEN b.event_time - INTERVAL '1' HOUR AND b.event_time + INTERVAL '1' HOUR
WHERE a.event_type = 'ORDER' AND b.event_type = 'ORDER'
GROUP BY a.sku, b.sku;
