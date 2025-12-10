/*---
name: stg_products
materialized: table
owner: data-platform
tags:
  - staging
  - products
tests:
  - unique: [product_id]
  - not_null: [product_id, product_name]
---*/

-- Staging model for products
-- Cleans and standardizes raw product data

SELECT 
    CAST(id AS INTEGER) AS product_id,
    TRIM(name) AS product_name,
    LOWER(TRIM(category)) AS category,
    CAST(price AS DECIMAL(10,2)) AS price
FROM raw_products
