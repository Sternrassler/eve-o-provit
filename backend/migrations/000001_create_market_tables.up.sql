-- Migration: Create market_orders and price_history tables
-- PostgreSQL Schema für dynamische Market-Daten (ESI)

-- Market Orders (aus ESI)
CREATE TABLE IF NOT EXISTS market_orders (
    order_id BIGINT PRIMARY KEY,
    type_id INTEGER NOT NULL,
    region_id INTEGER NOT NULL,
    system_id INTEGER,
    location_id BIGINT NOT NULL,
    is_buy_order BOOLEAN NOT NULL,
    price DECIMAL(20,2) NOT NULL,
    volume_remain INTEGER NOT NULL,
    volume_total INTEGER NOT NULL,
    min_volume INTEGER DEFAULT 1,
    duration INTEGER NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL,
    range VARCHAR(50),
    cached_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_market_orders_type_region ON market_orders(type_id, region_id);
CREATE INDEX IF NOT EXISTS idx_market_orders_cached ON market_orders(cached_at);
CREATE INDEX IF NOT EXISTS idx_market_orders_is_buy ON market_orders(is_buy_order);

COMMENT ON TABLE market_orders IS 'Market orders fetched from ESI API';
COMMENT ON COLUMN market_orders.cached_at IS 'Timestamp when order was cached from ESI';
COMMENT ON COLUMN market_orders.issued_at IS 'Timestamp when order was issued in EVE Online';

-- Price History (aggregiert)
CREATE TABLE IF NOT EXISTS price_history (
    id SERIAL PRIMARY KEY,
    type_id INTEGER NOT NULL,
    region_id INTEGER NOT NULL,
    date DATE NOT NULL,
    highest DECIMAL(19,2),
    lowest DECIMAL(19,2),
    average DECIMAL(19,2),
    volume BIGINT,
    order_count INTEGER,
    UNIQUE(type_id, region_id, date)
);

CREATE INDEX idx_price_history_lookup ON price_history(type_id, region_id, date DESC);

COMMENT ON TABLE price_history IS 'Aggregated historical price data from ESI';
COMMENT ON COLUMN price_history.date IS 'Date of aggregated data';
