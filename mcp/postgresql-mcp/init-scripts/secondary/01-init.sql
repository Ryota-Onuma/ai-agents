-- Secondary database initialization script (e.g., for inventory management)

-- Warehouses table
CREATE TABLE warehouses (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    location VARCHAR(200) NOT NULL,
    capacity INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Inventory table
CREATE TABLE inventory (
    id SERIAL PRIMARY KEY,
    warehouse_id INTEGER REFERENCES warehouses(id),
    product_id INTEGER NOT NULL, -- References products from primary DB
    quantity INTEGER NOT NULL DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Suppliers table
CREATE TABLE suppliers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    contact_email VARCHAR(100),
    phone VARCHAR(20),
    address TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample data
INSERT INTO warehouses (name, location, capacity) VALUES
    ('Main Warehouse', 'New York, NY', 10000),
    ('West Coast Hub', 'Los Angeles, CA', 8000),
    ('East Coast Hub', 'Miami, FL', 6000);

INSERT INTO inventory (warehouse_id, product_id, quantity) VALUES
    (1, 1, 25),  -- Laptop Pro at Main Warehouse
    (1, 2, 150), -- Wireless Mouse at Main Warehouse
    (1, 3, 50),  -- Mechanical Keyboard at Main Warehouse
    (2, 1, 15),  -- Laptop Pro at West Coast Hub
    (2, 2, 50),  -- Wireless Mouse at West Coast Hub
    (3, 2, 75),  -- Wireless Mouse at East Coast Hub
    (3, 3, 25);  -- Mechanical Keyboard at East Coast Hub

INSERT INTO suppliers (name, contact_email, phone, address) VALUES
    ('TechSupplier Inc', 'contact@techsupplier.com', '+1-555-0101', '123 Tech Street, Silicon Valley, CA'),
    ('Hardware Solutions', 'sales@hardwaresolutions.com', '+1-555-0202', '456 Component Ave, Austin, TX'),
    ('Global Electronics', 'info@globalelectronics.com', '+1-555-0303', '789 Circuit Blvd, Boston, MA');