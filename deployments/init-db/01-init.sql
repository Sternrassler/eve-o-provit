-- EVE-O-Provit Database Initialization
-- PostgreSQL Schema für Market Data & User Management

-- Enable Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For text search

-- Users & Authentication
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);

-- Market Orders (cached from ESI)
CREATE TABLE IF NOT EXISTS market_orders (
    order_id BIGINT PRIMARY KEY,
    type_id INT NOT NULL,
    region_id INT NOT NULL,
    system_id INT,
    location_id BIGINT NOT NULL,
    is_buy_order BOOLEAN NOT NULL,
    price NUMERIC(20, 2) NOT NULL,
    volume_remain INT NOT NULL,
    volume_total INT NOT NULL,
    min_volume INT DEFAULT 1,
    duration INT NOT NULL,
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL,
    range VARCHAR(50),
    cached_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_market_orders_type_region ON market_orders(type_id, region_id);
CREATE INDEX idx_market_orders_is_buy ON market_orders(is_buy_order);
CREATE INDEX idx_market_orders_cached ON market_orders(cached_at);

-- Price/Market History (aggregated). Table name MUST be price_history — that is
-- what the backend (MarketRepository.UpsertPriceHistory/GetVolumeHistory) and the
-- golang-migrate migration 000001 use. (A historical init-db named it market_history,
-- which the code never queried → volume/baseline silently returned 0 in prod.)
CREATE TABLE IF NOT EXISTS price_history (
    id SERIAL PRIMARY KEY,
    type_id INTEGER NOT NULL,
    region_id INTEGER NOT NULL,
    date DATE NOT NULL,
    highest DECIMAL(19, 2),
    lowest DECIMAL(19, 2),
    average DECIMAL(19, 2),
    volume BIGINT,
    order_count INTEGER,
    UNIQUE(type_id, region_id, date)
);

CREATE INDEX IF NOT EXISTS idx_price_history_lookup ON price_history(type_id, region_id, date DESC);

-- User Watchlists
CREATE TABLE IF NOT EXISTS watchlists (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_watchlists_user_id ON watchlists(user_id);

-- Watchlist Items
CREATE TABLE IF NOT EXISTS watchlist_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    watchlist_id UUID NOT NULL REFERENCES watchlists(id) ON DELETE CASCADE,
    type_id INT NOT NULL,
    buy_price NUMERIC(20, 2),
    sell_price NUMERIC(20, 2),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_watchlist_items_watchlist_id ON watchlist_items(watchlist_id);
CREATE INDEX idx_watchlist_items_type_id ON watchlist_items(type_id);

-- Profit Calculations Cache
CREATE TABLE IF NOT EXISTS profit_calculations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type_id INT NOT NULL,
    buy_region_id INT NOT NULL,
    sell_region_id INT NOT NULL,
    buy_price NUMERIC(20, 2) NOT NULL,
    sell_price NUMERIC(20, 2) NOT NULL,
    profit_per_unit NUMERIC(20, 2) NOT NULL,
    profit_margin NUMERIC(5, 2),
    volume_available INT,
    calculated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_profit_calculations_type ON profit_calculations(type_id);
CREATE INDEX idx_profit_calculations_margin ON profit_calculations(profit_margin DESC);
CREATE INDEX idx_profit_calculations_calculated ON profit_calculations(calculated_at);

-- Competition tracking for Multi-Hub Comparison (#43)
-- Mirrors backend/migrations/000002_competition_tracking.up.sql for the prod
-- init-db apply path (HETZNER-DEPLOY Step 5). Idempotent.
CREATE TABLE IF NOT EXISTS competition_tracked (
    type_id        INT NOT NULL,
    region_id      INT NOT NULL,
    last_requested TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (type_id, region_id)
);
CREATE INDEX IF NOT EXISTS idx_competition_tracked_last_requested ON competition_tracked(last_requested);

CREATE TABLE IF NOT EXISTS competition_snapshot (
    id          BIGSERIAL PRIMARY KEY,
    type_id     INT NOT NULL,
    region_id   INT NOT NULL,
    taken_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    fingerprint JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_competition_snapshot_lookup ON competition_snapshot(type_id, region_id, taken_at DESC);

CREATE TABLE IF NOT EXISTS competition_metric (
    type_id          INT NOT NULL,
    region_id        INT NOT NULL,
    changes_per_hour DOUBLE PRECISION NOT NULL DEFAULT 0,
    window_start     TIMESTAMP WITH TIME ZONE,
    window_end       TIMESTAMP WITH TIME ZONE,
    source           VARCHAR(16) NOT NULL DEFAULT 'baseline',
    updated_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (type_id, region_id)
);

-- Comments
COMMENT ON TABLE users IS 'User accounts for EVE-O-Provit';
COMMENT ON TABLE market_orders IS 'Cached market orders from ESI API';
COMMENT ON TABLE price_history IS 'Aggregated historical market data (volume/order_count) from ESI';
COMMENT ON TABLE watchlists IS 'User-defined item watchlists';
COMMENT ON TABLE profit_calculations IS 'Cached profit margin calculations';

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO eveprovit;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO eveprovit;
