CREATE TABLE IF NOT EXISTS orders (
                                      order_id UUID PRIMARY KEY,
                                      user_id UUID NOT NULL,
                                      market_id TEXT NOT NULL,
                                      order_type SMALLINT NOT NULL, -- 0: UNKNOWN, 1: BUY, 2: SELL
                                      status SMALLINT NOT NULL,      -- 0: PENDING, 1: FILLED, 2: CANCELED
                                      price NUMERIC(20, 8) NOT NULL, -- Храним как число, в Go конвертируем в string
    quantity NUMERIC(20, 8) NOT NULL,
    filled_quantity NUMERIC(20, 8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );


CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_market_id ON orders(market_id);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC);