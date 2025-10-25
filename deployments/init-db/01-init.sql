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

-- Market History (aggregated)
CREATE TABLE IF NOT EXISTS market_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type_id INT NOT NULL,
    region_id INT NOT NULL,
    date DATE NOT NULL,
    average NUMERIC(20, 2),
    highest NUMERIC(20, 2),
    lowest NUMERIC(20, 2),
    volume BIGINT,
    order_count INT,
    UNIQUE(type_id, region_id, date)
);

CREATE INDEX idx_market_history_type_region_date ON market_history(type_id, region_id, date DESC);

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

-- Comments
COMMENT ON TABLE users IS 'User accounts for EVE-O-Provit';
COMMENT ON TABLE market_orders IS 'Cached market orders from ESI API';
COMMENT ON TABLE market_history IS 'Historical market data for trend analysis';
COMMENT ON TABLE watchlists IS 'User-defined item watchlists';
COMMENT ON TABLE profit_calculations IS 'Cached profit margin calculations';

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO eveprovit;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO eveprovit;
