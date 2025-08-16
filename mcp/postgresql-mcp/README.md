# PostgreSQL MCP Server

A Model Context Protocol (MCP) server for PostgreSQL database operations written in Go. This server provides seamless access to multiple PostgreSQL databases with CRUD operations through the MCP protocol.

## Features

- 🗄️ **Multiple Database Support**: Connect to multiple PostgreSQL databases simultaneously
- 🔄 **CRUD Operations**: Complete Create, Read, Update, Delete functionality
- 🐳 **Docker Ready**: Comes with Docker Compose configuration for easy setup
- 🔍 **Schema Introspection**: List tables and describe table structures
- 🔧 **Connection Management**: Easy connection switching and management
- ⚡ **High Performance**: Written in Go for optimal performance
- 🔒 **SQL Injection Protection**: Parameterized queries with identifier validation

## Quick Start

### 1. Setup Environment

```bash
# Clone or navigate to the project
cd mcp/postgresql-mcp

# Install Go dependencies
make deps

# Build the project
make build
```

### 2. Start PostgreSQL with Docker Compose

```bash
# Start the databases
docker-compose up -d

# Verify databases are running
docker-compose ps
```

### 3. Run the MCP Server

```bash
# Run the server directly
make run

# Or run the built binary
./bin/postgresql-mcp
```

### 4. Test the Connection

The server supports environment variable for database URLs:
- `POSTGRESQL_URLS`: Comma-separated list of PostgreSQL connection URLs

All URLs are validated on startup and **must resolve to local hosts only** (`localhost`, `127.0.0.0/8`, `::1`, or allowed Unix sockets). If any URL points to a remote host, the server exits with an error before attempting connection. Existing `.mcp.json` configurations continue to work without modification.

The server will automatically extract database names from the URLs and create named connections.

## Available Tools

### Connection Management

- **connect_database**: Connect to a new PostgreSQL database
  ```json
  {
    "name": "my_db",
    "connection_string": "postgresql://user:pass@host:port/database"
  }
  ```

- **disconnect_database**: Close and remove a connection by name
  ```json
  {
    "name": "my_db"
  }
  ```

- **list_connections**: List all active database connections

### Query Operations

- **query**: Execute SELECT queries safely
  ```json
  {
    "sql": "SELECT * FROM users WHERE id = $1",
    "params": ["1"],
    "database": "primary_db",
    "limit": 10,
    "statement_timeout_ms": 5000
  }
  ```

### CRUD Operations

- **insert**: INSERT with validated identifiers
  ```json
  {
    "table": "users",
    "data": {"username": "newuser", "email": "user@example.com"},
    "database": "primary_db",
    "returning": true,
    "statement_timeout_ms": 5000
  }
  ```

- **update**: UPDATE with validated identifiers and WHERE map
  ```json
  {
    "table": "users", 
    "data": {"email": "newemail@example.com"},
    "where": {"id": 1},
    "database": "primary_db",
    "returning": false,
    "statement_timeout_ms": 5000
  }
  ```

- **delete**: DELETE with validated identifiers and WHERE map
  ```json
  {
    "table": "users",
    "where": {"id": 1},
    "database": "primary_db",
    "returning": false,
    "statement_timeout_ms": 5000
  }
  ```

### Schema Operations

- **list_schemas**: List non-system schemas
  ```json
  {
    "database": "primary_db"
  }
  ```

- **list_tables**: List tables under a schema
  ```json
  {
    "database": "primary_db",
    "schema": "public"
  }
  ```

- **describe_table**: Describe columns for a table
  ```json
  {
    "table": "users",
    "database": "primary_db"
  }
  ```

## Environment Configuration

Set the `POSTGRESQL_URLS` environment variable with comma-separated database URLs:

```bash
# Set environment variable for multiple databases
export POSTGRESQL_URLS="postgresql://postgres:password123@localhost:5432/primary_db,postgresql://postgres:password123@localhost:5433/secondary_db,postgresql://postgres:password123@localhost:5434/analytics_db"

# Or run with the variable
POSTGRESQL_URLS="postgresql://postgres:password123@localhost:5432/primary_db" ./bin/postgresql-mcp
```

The server automatically extracts database names from the URLs and creates named connections (e.g., `primary_db`, `secondary_db`, `analytics_db`).

> **Note**: Connections forwarded through SSH tunnels to `localhost` cannot be detected by this validation.

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

```bash
# Connect to a database
mcp__postgresql__connect_database name="my_app" connection_string="postgresql://user:pass@host:port/db"

# Query data
mcp__postgresql__query sql="SELECT * FROM users LIMIT 5" database="my_app"

# Insert data
mcp__postgresql__insert table="users" data='{"username": "alice", "email": "alice@example.com"}' database="my_app"

# Update data  
mcp__postgresql__update table="users" data='{"email": "alice.new@example.com"}' where='{"id": 1}' database="my_app"

# List tables
mcp__postgresql__list_tables database="my_app"

# Describe a table
mcp__postgresql__describe_table table="users" database="my_app"
```

## Integration with Claude Code

To use this MCP server with Claude Code, add it to your MCP configuration:

```json
{
  "mcpServers": {
    "postgresql": {
      "command": "/absolute/path/to/mcp/postgresql-mcp/bin/postgresql-mcp",
      "env": {
        "POSTGRESQL_URLS": "postgresql://postgres:password123@localhost:5432/primary_db,postgresql://postgres:password123@localhost:5433/secondary_db"
      }
    }
  }
}
```

Notes:
- Replace `/absolute/path/to/...` with the actual path on your machine.
- Alternatively, use `make run` from `mcp/postgresql-mcp` during development.

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
make deps

# Run with debug logging
export MCP_LOG_LEVEL=debug
make run
# or
./bin/postgresql-mcp

# Run tests
go test ./...
```

## License

MIT License - see LICENSE file for details.
