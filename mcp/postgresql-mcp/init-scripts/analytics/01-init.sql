-- Analytics PostgreSQL instance initialization script
-- This instance will host multiple databases for different analytics purposes

-- Create additional databases
CREATE DATABASE reporting_db;
CREATE DATABASE monitoring_db;
CREATE DATABASE metrics_db;

-- Connect to analytics_db (default database)
\c analytics_db;

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

-- Initialize reporting_db
\c reporting_db;

CREATE TABLE report_schedules (
    id SERIAL PRIMARY KEY,
    report_name VARCHAR(100) NOT NULL,
    description TEXT,
    schedule_cron VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE report_executions (
    id SERIAL PRIMARY KEY,
    report_schedule_id INTEGER REFERENCES report_schedules(id),
    execution_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) DEFAULT 'pending',
    file_path VARCHAR(500),
    error_message TEXT
);

INSERT INTO report_schedules (report_name, description, schedule_cron) VALUES
    ('Daily Sales Report', 'Daily sales summary with key metrics', '0 9 * * *'),
    ('Weekly User Activity', 'Weekly user engagement and activity report', '0 10 * * 1'),
    ('Monthly Inventory Report', 'Monthly inventory levels and turnover', '0 8 1 * *');

-- Initialize monitoring_db
\c monitoring_db;

CREATE TABLE system_metrics (
    id SERIAL PRIMARY KEY,
    metric_name VARCHAR(50) NOT NULL,
    metric_value DECIMAL(15,4),
    unit VARCHAR(20),
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    source VARCHAR(50)
);

CREATE TABLE alert_rules (
    id SERIAL PRIMARY KEY,
    rule_name VARCHAR(100) NOT NULL,
    metric_name VARCHAR(50) NOT NULL,
    condition_operator VARCHAR(10), -- >, <, =, >=, <=
    threshold_value DECIMAL(15,4),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO system_metrics (metric_name, metric_value, unit, source) VALUES
    ('cpu_usage_percent', 45.2, 'percent', 'server-01'),
    ('memory_usage_percent', 68.5, 'percent', 'server-01'),
    ('disk_usage_percent', 78.1, 'percent', 'server-01'),
    ('response_time_ms', 120.5, 'milliseconds', 'api-gateway'),
    ('error_rate_percent', 0.8, 'percent', 'application');

INSERT INTO alert_rules (rule_name, metric_name, condition_operator, threshold_value) VALUES
    ('High CPU Usage', 'cpu_usage_percent', '>', 80.0),
    ('High Memory Usage', 'memory_usage_percent', '>', 85.0),
    ('Slow Response Time', 'response_time_ms', '>', 500.0),
    ('High Error Rate', 'error_rate_percent', '>', 5.0);

-- Initialize metrics_db
\c metrics_db;

CREATE TABLE business_kpis (
    id SERIAL PRIMARY KEY,
    kpi_name VARCHAR(100) NOT NULL,
    kpi_value DECIMAL(15,4),
    period_start DATE,
    period_end DATE,
    category VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE growth_metrics (
    id SERIAL PRIMARY KEY,
    metric_type VARCHAR(50) NOT NULL, -- revenue, users, orders, etc.
    current_period_value DECIMAL(15,4),
    previous_period_value DECIMAL(15,4),
    growth_rate_percent DECIMAL(8,4),
    period_type VARCHAR(20), -- daily, weekly, monthly, yearly
    period_date DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO business_kpis (kpi_name, kpi_value, period_start, period_end, category) VALUES
    ('Monthly Recurring Revenue', 45000.00, '2024-01-01', '2024-01-31', 'revenue'),
    ('Customer Acquisition Cost', 25.50, '2024-01-01', '2024-01-31', 'marketing'),
    ('Customer Lifetime Value', 450.00, '2024-01-01', '2024-01-31', 'revenue'),
    ('Monthly Active Users', 12500, '2024-01-01', '2024-01-31', 'engagement');

INSERT INTO growth_metrics (metric_type, current_period_value, previous_period_value, growth_rate_percent, period_type, period_date) VALUES
    ('revenue', 45000.00, 42000.00, 7.14, 'monthly', '2024-01-31'),
    ('users', 12500, 11800, 5.93, 'monthly', '2024-01-31'),
    ('orders', 850, 790, 7.59, 'monthly', '2024-01-31');