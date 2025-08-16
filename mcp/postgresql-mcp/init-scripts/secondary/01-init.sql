-- Secondary database initialization script with multiple schemas

-- Create multiple schemas for different business domains
CREATE SCHEMA IF NOT EXISTS inventory;
CREATE SCHEMA IF NOT EXISTS logistics;
CREATE SCHEMA IF NOT EXISTS finance;

-- Set default schema for initial setup
SET search_path TO inventory, public;

-- INVENTORY SCHEMA TABLES
-- Warehouses table
CREATE TABLE inventory.warehouses (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    location VARCHAR(200) NOT NULL,
    capacity INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Inventory table
CREATE TABLE inventory.stock (
    id SERIAL PRIMARY KEY,
    warehouse_id INTEGER REFERENCES inventory.warehouses(id),
    product_id INTEGER NOT NULL, -- References products from primary DB
    quantity INTEGER NOT NULL DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Suppliers table
CREATE TABLE inventory.suppliers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    contact_email VARCHAR(100),
    phone VARCHAR(20),
    address TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- LOGISTICS SCHEMA TABLES
CREATE TABLE logistics.shipping_companies (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    contact_email VARCHAR(100),
    tracking_url_pattern VARCHAR(200),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE logistics.shipments (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL, -- References orders from primary DB
    shipping_company_id INTEGER REFERENCES logistics.shipping_companies(id),
    tracking_number VARCHAR(100),
    status VARCHAR(50) DEFAULT 'pending',
    shipped_at TIMESTAMP,
    delivered_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE logistics.delivery_routes (
    id SERIAL PRIMARY KEY,
    warehouse_id INTEGER, -- References inventory.warehouses(id)
    region VARCHAR(100) NOT NULL,
    estimated_days INTEGER DEFAULT 3,
    cost_per_km DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- FINANCE SCHEMA TABLES
CREATE TABLE finance.payment_methods (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    processor VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE finance.transactions (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL, -- References orders from primary DB
    payment_method_id INTEGER REFERENCES finance.payment_methods(id),
    amount DECIMAL(12,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(20) DEFAULT 'pending',
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE finance.refunds (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER REFERENCES finance.transactions(id),
    amount DECIMAL(12,2) NOT NULL,
    reason TEXT,
    status VARCHAR(20) DEFAULT 'pending',
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample data for INVENTORY schema
INSERT INTO inventory.warehouses (name, location, capacity) VALUES
    ('Main Warehouse', 'New York, NY', 10000),
    ('West Coast Hub', 'Los Angeles, CA', 8000),
    ('East Coast Hub', 'Miami, FL', 6000),
    ('European Distribution Center', 'Hamburg, Germany', 12000);

INSERT INTO inventory.stock (warehouse_id, product_id, quantity) VALUES
    (1, 1, 25),  -- Laptop Pro at Main Warehouse
    (1, 2, 150), -- Wireless Mouse at Main Warehouse
    (1, 3, 50),  -- Mechanical Keyboard at Main Warehouse
    (2, 1, 15),  -- Laptop Pro at West Coast Hub
    (2, 2, 50),  -- Wireless Mouse at West Coast Hub
    (3, 2, 75),  -- Wireless Mouse at East Coast Hub
    (3, 3, 25),  -- Mechanical Keyboard at East Coast Hub
    (4, 1, 30),  -- Laptop Pro at European DC
    (4, 2, 200), -- Wireless Mouse at European DC
    (4, 3, 80);  -- Mechanical Keyboard at European DC

INSERT INTO inventory.suppliers (name, contact_email, phone, address) VALUES
    ('TechSupplier Inc', 'contact@techsupplier.com', '+1-555-0101', '123 Tech Street, Silicon Valley, CA'),
    ('Hardware Solutions', 'sales@hardwaresolutions.com', '+1-555-0202', '456 Component Ave, Austin, TX'),
    ('Global Electronics', 'info@globalelectronics.com', '+1-555-0303', '789 Circuit Blvd, Boston, MA'),
    ('European Tech GmbH', 'sales@eurotech.de', '+49-30-12345678', 'Unter den Linden 1, Berlin, Germany');

-- Insert sample data for LOGISTICS schema
INSERT INTO logistics.shipping_companies (name, contact_email, tracking_url_pattern) VALUES
    ('FedEx', 'support@fedex.com', 'https://www.fedex.com/fedextrack/?tracknumbers={}'),
    ('UPS', 'support@ups.com', 'https://www.ups.com/track?tracknum={}'),
    ('DHL', 'support@dhl.com', 'https://www.dhl.com/tracking?trackingNumber={}'),
    ('USPS', 'support@usps.com', 'https://tools.usps.com/go/TrackConfirmAction?qtc_tLabels1={}');

INSERT INTO logistics.shipments (order_id, shipping_company_id, tracking_number, status) VALUES
    (1, 1, 'FDX123456789', 'in_transit'),
    (2, 2, 'UPS987654321', 'delivered'),
    (3, 3, 'DHL555666777', 'pending'),
    (4, 1, 'FDX111222333', 'shipped'),
    (5, 4, 'USPS444555666', 'in_transit');

INSERT INTO logistics.delivery_routes (warehouse_id, region, estimated_days, cost_per_km) VALUES
    (1, 'Northeast US', 2, 0.50),
    (1, 'Southeast US', 3, 0.45),
    (2, 'West Coast US', 1, 0.55),
    (2, 'Mountain West US', 2, 0.48),
    (3, 'Southeast US', 1, 0.52),
    (4, 'Western Europe', 2, 0.65),
    (4, 'Eastern Europe', 4, 0.40);

-- Insert sample data for FINANCE schema
INSERT INTO finance.payment_methods (name, processor, is_active) VALUES
    ('Credit Card', 'Stripe', true),
    ('PayPal', 'PayPal', true),
    ('Bank Transfer', 'Plaid', true),
    ('Apple Pay', 'Stripe', true),
    ('Google Pay', 'Stripe', true),
    ('Cryptocurrency', 'Coinbase', false);

INSERT INTO finance.transactions (order_id, payment_method_id, amount, currency, status) VALUES
    (1, 1, 1299.99, 'USD', 'completed'),
    (2, 2, 29.99, 'USD', 'completed'),
    (3, 1, 89.99, 'USD', 'pending'),
    (4, 3, 2599.98, 'USD', 'completed'),
    (5, 4, 29.99, 'USD', 'completed'),
    (6, 1, 1379.97, 'EUR', 'completed');

INSERT INTO finance.refunds (transaction_id, amount, reason, status) VALUES
    (2, 29.99, 'Product defective', 'completed'),
    (6, 89.99, 'Customer changed mind', 'pending');