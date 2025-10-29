-- Rollback migration for market tables

DROP INDEX IF EXISTS idx_price_history_lookup;
DROP TABLE IF EXISTS price_history;

DROP INDEX IF EXISTS idx_market_orders_location;
DROP INDEX IF EXISTS idx_market_orders_fetched;
DROP INDEX IF EXISTS idx_market_orders_type_region;
DROP TABLE IF EXISTS market_orders;
