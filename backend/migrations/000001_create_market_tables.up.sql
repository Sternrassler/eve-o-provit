-- Migration: Create market_orders and price_history tables
-- PostgreSQL Schema für dynamische Market-Daten (ESI)

-- Market Orders (aus ESI)
CREATE TABLE IF NOT EXISTS market_orders (
    order_id BIGINT PRIMARY KEY,
    type_id INTEGER NOT NULL,
    region_id INTEGER NOT NULL,
    location_id BIGINT NOT NULL,
    is_buy_order BOOLEAN NOT NULL,
    price DECIMAL(19,2) NOT NULL,
    volume_total INTEGER NOT NULL,
    volume_remain INTEGER NOT NULL,
    min_volume INTEGER,
    issued TIMESTAMPTZ NOT NULL,
    duration INTEGER NOT NULL,
    fetched_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(order_id, fetched_at)
);

CREATE INDEX idx_market_orders_type_region ON market_orders(type_id, region_id);
CREATE INDEX idx_market_orders_fetched ON market_orders(fetched_at);
CREATE INDEX idx_market_orders_location ON market_orders(location_id);

COMMENT ON TABLE market_orders IS 'Market orders fetched from ESI API';
COMMENT ON COLUMN market_orders.fetched_at IS 'Timestamp when order was fetched from ESI';

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
