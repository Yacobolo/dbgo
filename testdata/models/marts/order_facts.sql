/*---
name: order_facts
materialized: incremental
unique_key: order_id
owner: analytics
schema: marts
tags:
  - marts
  - orders
  - incremental
tests:
  - unique: [order_id]
  - not_null: [order_id, customer_id, order_date]
---*/

-- Incremental order facts - only processes new orders
-- Note: Incremental WHERE clause is handled by the engine based on unique_key

SELECT 
    o.order_id,
    o.customer_id,
    c.customer_name,
    o.product_id,
    o.quantity,
    o.unit_price,
    o.order_total,
    o.order_date,
    o.status,
    CURRENT_TIMESTAMP AS processed_at
FROM staging.stg_orders o
JOIN staging.stg_customers c ON o.customer_id = c.customer_id
