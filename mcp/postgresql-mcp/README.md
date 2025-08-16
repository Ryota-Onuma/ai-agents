# PostgreSQL MCP Server

A Model Context Protocol (MCP) server for PostgreSQL database operations. This server provides seamless access to multiple PostgreSQL databases with CRUD operations through the MCP protocol.

## Features

- 🗄️ **Multiple Database Support**: Connect to multiple PostgreSQL databases simultaneously
- 🔄 **CRUD Operations**: Complete Create, Read, Update, Delete functionality
- 🐳 **Docker Ready**: Comes with Docker Compose configuration for easy setup
- 🔍 **Schema Introspection**: List tables and describe table structures
- 🔧 **Connection Management**: Easy connection switching and management
- 📊 **Resource Support**: MCP resource support for connection information

## Quick Start

### 1. Setup Environment

```bash
# Clone or create the project
cd mcp/postgresql-mcp

# Install dependencies
pip install -r requirements.txt

# Copy and configure environment
cp .env.example .env
# Edit .env with your database connection strings
```

### 2. Start PostgreSQL with Docker Compose

```bash
# Start the databases
docker-compose -f docker-compose.example.yml up -d

# Verify databases are running
docker-compose -f docker-compose.example.yml ps
```

### 3. Run the MCP Server

```bash
# Run the server
python main.py
```

### 4. Test the Connection

The server will automatically connect to databases specified in environment variables:
- `POSTGRESQL_DEFAULT_URL`
- `POSTGRESQL_PRIMARY_URL` 
- `POSTGRESQL_SECONDARY_URL`
- `POSTGRESQL_ANALYTICS_URL`

## Available Tools

### Connection Management

- **connect_database**: Connect to a new PostgreSQL database
  ```json
  {
    "name": "my_db",
    "connection_string": "postgresql://user:pass@host:port/database"
  }
  ```

- **list_connections**: List all active database connections

### Query Operations

- **query**: Execute SELECT queries
  ```json
  {
    "sql": "SELECT * FROM users WHERE id = $1",
    "params": ["1"],
    "database": "primary"
  }
  ```

### CRUD Operations

- **insert**: Insert new records
  ```json
  {
    "table": "users",
    "data": {"username": "newuser", "email": "user@example.com"},
    "database": "primary"
  }
  ```

- **update**: Update existing records
  ```json
  {
    "table": "users", 
    "data": {"email": "newemail@example.com"},
    "where": {"id": 1},
    "database": "primary"
  }
  ```

- **delete**: Delete records
  ```json
  {
    "table": "users",
    "where": {"id": 1},
    "database": "primary"
  }
  ```

### Schema Operations

- **list_tables**: List all tables in a database
- **describe_table**: Get detailed table schema information

## Environment Configuration

Create a `.env` file with your database connections:

```bash
# Default connection (used when no database is specified)
POSTGRESQL_DEFAULT_URL=postgresql://postgres:password123@localhost:5432/primary_db

# Named connections for different services
POSTGRESQL_PRIMARY_URL=postgresql://postgres:password123@localhost:5432/primary_db
POSTGRESQL_SECONDARY_URL=postgresql://postgres:password123@localhost:5433/secondary_db
POSTGRESQL_ANALYTICS_URL=postgresql://postgres:password123@localhost:5434/analytics_db
```

## Database Architecture Example

The provided Docker Compose setup creates three databases:

1. **Primary DB** (port 5432): Main application data
   - Users, Products, Orders tables
   
2. **Secondary DB** (port 5433): Inventory management
   - Warehouses, Inventory, Suppliers tables
   
3. **Analytics DB** (port 5434): Analytics and reporting
   - User events, Product views, Sales metrics, Search queries

## Usage in Claude Code

Once the MCP server is running, you can use it in Claude Code:

```
# Connect to a database
connect_database name="my_app" connection_string="postgresql://user:pass@host:port/db"

# Query data
query sql="SELECT * FROM users LIMIT 5" database="my_app"

# Insert data
insert table="users" data='{"username": "alice", "email": "alice@example.com"}' database="my_app"

# Update data  
update table="users" data='{"email": "alice.new@example.com"}' where='{"id": 1}' database="my_app"

# List tables
list_tables database="my_app"

# Describe a table
describe_table table="users" database="my_app"
```

## Integration with Claude Code

To use this MCP server with Claude Code, add it to your MCP configuration:

```json
{
  "mcpServers": {
    "postgresql": {
      "command": "python",
      "args": ["/path/to/mcp/postgresql-mcp/main.py"],
      "env": {
        "POSTGRESQL_DEFAULT_URL": "postgresql://postgres:password123@localhost:5432/primary_db",
        "POSTGRESQL_PRIMARY_URL": "postgresql://postgres:password123@localhost:5432/primary_db",
        "POSTGRESQL_SECONDARY_URL": "postgresql://postgres:password123@localhost:5433/secondary_db"
      }
    }
  }
}
```

## Security Considerations

- Always use connection strings with appropriate authentication
- Consider using connection pooling for production environments
- Restrict database permissions to minimum required access
- Use environment variables for sensitive connection information
- Enable SSL/TLS for production database connections

## Troubleshooting

### Connection Issues
- Verify database is running: `docker-compose ps`
- Check connection string format
- Ensure database allows external connections
- Verify firewall/network settings

### Permission Errors
- Ensure database user has necessary permissions
- Check table/schema access rights
- Verify SSL requirements if applicable

### Performance
- Use parameterized queries to prevent SQL injection
- Consider connection pooling for high-traffic scenarios
- Monitor query performance and add indexes as needed

## Development

To contribute or modify the server:

```bash
# Install development dependencies
pip install -r requirements.txt

# Run with debug logging
export MCP_LOG_LEVEL=debug
python main.py

# Run tests (if available)
python -m pytest
```

## License

MIT License - see LICENSE file for details.