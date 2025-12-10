/*---
name: stg_customers
materialized: table
owner: data-platform
tags:
  - staging
  - customers
tests:
  - unique: [customer_id]
  - not_null: [customer_id, email]
---*/

-- Staging model for customers
-- Cleans and standardizes raw customer data

SELECT 
    {{ utils.safe_cast("id", "INTEGER") }} AS customer_id,
    {{ utils.upper("TRIM(email)") }} AS email,
    TRIM(name) AS customer_name,
    {{ utils.safe_cast("created_at", "DATE") }} AS created_at,
    {{ utils.coalesce("is_active", "false") }} AS is_active
FROM raw_customers
