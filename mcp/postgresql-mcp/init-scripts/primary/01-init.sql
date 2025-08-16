-- Primary database initialization script

-- Users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Products table
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    category_id INTEGER,
    stock_quantity INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Orders table
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    total_amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample data
INSERT INTO users (username, email, password_hash) VALUES
    ('admin', 'admin@example.com', 'hashed_password_1'),
    ('john_doe', 'john@example.com', 'hashed_password_2'),
    ('jane_smith', 'jane@example.com', 'hashed_password_3');

INSERT INTO products (name, description, price, category_id, stock_quantity) VALUES
    ('Laptop Pro', 'High-performance laptop', 1299.99, 1, 50),
    ('Wireless Mouse', 'Ergonomic wireless mouse', 29.99, 2, 200),
    ('Mechanical Keyboard', 'RGB mechanical keyboard', 129.99, 2, 75);

INSERT INTO orders (user_id, total_amount, status) VALUES
    (1, 1329.98, 'completed'),
    (2, 29.99, 'pending'),
    (3, 259.98, 'processing');