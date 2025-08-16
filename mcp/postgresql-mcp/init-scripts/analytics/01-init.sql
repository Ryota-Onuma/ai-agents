-- Analytics database initialization script

-- User events table for tracking user behavior
CREATE TABLE user_events (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL, -- References users from primary DB
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    session_id VARCHAR(100)
);

-- Product views table
CREATE TABLE product_views (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL, -- References products from primary DB
    user_id INTEGER, -- References users from primary DB (nullable for anonymous views)
    view_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    referrer VARCHAR(500),
    user_agent TEXT
);

-- Sales metrics aggregation table
CREATE TABLE daily_sales_metrics (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    total_orders INTEGER DEFAULT 0,
    total_revenue DECIMAL(12,2) DEFAULT 0,
    average_order_value DECIMAL(10,2) DEFAULT 0,
    new_customers INTEGER DEFAULT 0,
    returning_customers INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Search queries table
CREATE TABLE search_queries (
    id SERIAL PRIMARY KEY,
    user_id INTEGER, -- References users from primary DB (nullable for anonymous searches)
    query TEXT NOT NULL,
    results_count INTEGER,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    clicked_product_id INTEGER -- References products from primary DB
);

-- Insert sample analytics data
INSERT INTO user_events (user_id, event_type, event_data, session_id) VALUES
    (1, 'login', '{"source": "web"}', 'sess_123456'),
    (1, 'view_product', '{"product_id": 1, "category": "laptops"}', 'sess_123456'),
    (1, 'add_to_cart', '{"product_id": 1, "quantity": 1}', 'sess_123456'),
    (1, 'purchase', '{"order_id": 1, "amount": 1329.98}', 'sess_123456'),
    (2, 'login', '{"source": "mobile"}', 'sess_789012'),
    (2, 'view_product', '{"product_id": 2, "category": "accessories"}', 'sess_789012');

INSERT INTO product_views (product_id, user_id, referrer, user_agent) VALUES
    (1, 1, 'https://google.com', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'),
    (1, 2, 'direct', 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_7_1 like Mac OS X)'),
    (2, 1, 'https://google.com', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'),
    (2, NULL, 'direct', 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36'),
    (3, 3, 'https://bing.com', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36');

INSERT INTO daily_sales_metrics (date, total_orders, total_revenue, average_order_value, new_customers, returning_customers) VALUES
    ('2024-01-15', 5, 2149.95, 429.99, 2, 3),
    ('2024-01-16', 8, 3299.92, 412.49, 1, 7),
    ('2024-01-17', 12, 4899.88, 408.32, 3, 9);

INSERT INTO search_queries (user_id, query, results_count, clicked_product_id) VALUES
    (1, 'laptop', 3, 1),
    (2, 'wireless mouse', 1, 2),
    (3, 'keyboard rgb', 1, 3),
    (NULL, 'gaming setup', 2, NULL),
    (1, 'mechanical keyboard', 1, 3);