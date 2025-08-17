package main

import (
	"flag"
	"fmt"
	"os"

	_ "github.com/lib/pq"

	"github.com/Ryota-Onuma/ai-agents/mcp/postgresql-mcp/internal/dbguard"
)

func main() {
	flag.Parse()

	// Get DSN string from command line arguments
	var dbURLs string
	if len(flag.Args()) > 0 {
		dbURLs = flag.Args()[0]
	}

	server := NewMCPServer("postgresql-mcp", "1.0.0")

	// Register tools
	server.AddTool("connect_database", "Connect to a PostgreSQL database", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Connection name",
			},
			"connection_string": map[string]interface{}{
				"type":        "string",
				"description": "PostgreSQL connection string",
			},
		},
		"required": []string{"name", "connection_string"},
	}, connectHandler)

	server.AddTool("disconnect_database", "Close and remove a connection by name", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Connection name",
			},
		},
		"required": []string{"name"},
	}, disconnectHandler)

	server.AddTool("list_connections", "List all connection names", map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}, listConnectionsHandler)

	server.AddTool("query", "Execute a SELECT query safely", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sql": map[string]interface{}{
				"type":        "string",
				"description": "SQL query to execute",
			},
			"params": map[string]interface{}{
				"type":        "array",
				"description": "Query parameters",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"database": map[string]interface{}{
				"type":        "string",
				"description": "Database connection name",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Limit number of results",
			},
			"statement_timeout_ms": map[string]interface{}{
				"type":        "integer",
				"description": "Statement timeout in milliseconds",
			},
		},
		"required": []string{"sql"},
	}, queryHandler)

	server.AddTool("insert", "INSERT with validated identifiers", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"table": map[string]interface{}{
				"type":        "string",
				"description": "Table name",
			},
			"data": map[string]interface{}{
				"type":        "object",
				"description": "Data to insert",
			},
			"database": map[string]interface{}{
				"type":        "string",
				"description": "Database connection name",
			},
			"returning": map[string]interface{}{
				"type":        "boolean",
				"description": "Return inserted rows",
			},
			"statement_timeout_ms": map[string]interface{}{
				"type":        "integer",
				"description": "Statement timeout in milliseconds",
			},
		},
		"required": []string{"table", "data"},
	}, insertHandler)

	server.AddTool("update", "UPDATE with validated identifiers and WHERE map", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"table": map[string]interface{}{
				"type":        "string",
				"description": "Table name",
			},
			"data": map[string]interface{}{
				"type":        "object",
				"description": "Data to update",
			},
			"where": map[string]interface{}{
				"type":        "object",
				"description": "WHERE conditions",
			},
			"database": map[string]interface{}{
				"type":        "string",
				"description": "Database connection name",
			},
			"returning": map[string]interface{}{
				"type":        "boolean",
				"description": "Return updated rows",
			},
			"statement_timeout_ms": map[string]interface{}{
				"type":        "integer",
				"description": "Statement timeout in milliseconds",
			},
		},
		"required": []string{"table", "data", "where"},
	}, updateHandler)

	server.AddTool("delete", "DELETE with validated identifiers and WHERE map", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"table": map[string]interface{}{
				"type":        "string",
				"description": "Table name",
			},
			"where": map[string]interface{}{
				"type":        "object",
				"description": "WHERE conditions",
			},
			"database": map[string]interface{}{
				"type":        "string",
				"description": "Database connection name",
			},
			"returning": map[string]interface{}{
				"type":        "boolean",
				"description": "Return deleted rows",
			},
			"statement_timeout_ms": map[string]interface{}{
				"type":        "integer",
				"description": "Statement timeout in milliseconds",
			},
		},
		"required": []string{"table", "where"},
	}, deleteHandler)

	server.AddTool("list_schemas", "List non-system schemas", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"database": map[string]interface{}{
				"type":        "string",
				"description": "Database connection name",
			},
		},
	}, listSchemasHandler)

	server.AddTool("list_tables", "List tables under a schema", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"database": map[string]interface{}{
				"type":        "string",
				"description": "Database connection name",
			},
			"schema": map[string]interface{}{
				"type":        "string",
				"description": "Schema name (default: public)",
			},
		},
	}, listTablesHandler)

	server.AddTool("describe_table", "Describe columns for a table", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"table": map[string]interface{}{
				"type":        "string",
				"description": "Table name (optionally with schema)",
			},
			"database": map[string]interface{}{
				"type":        "string",
				"description": "Database connection name",
			},
		},
		"required": []string{"table"},
	}, describeTableHandler)

	// Auto-connect via command line arguments with local-only enforcement
	var urls []string
	var err error

	if dbURLs == "" {
		fmt.Println("no database URLs provided: pass DSN string as argument")
		os.Exit(1)
	}

	urls, err = dbguard.LoadPostgresURLsFromArgs(dbURLs)
	if err != nil {
		fmt.Println("invalid database URLs:", err)
		os.Exit(1)
	}
	dsns, err := dbguard.EnforceLocalForURLs(urls)
	if err != nil {
		fmt.Println("database URL rejected:", err)
		os.Exit(1)
	}
	for _, dsn := range dsns {
		name := extractDatabaseName(dsn)
		if name == "" {
			logger.Printf("Failed to extract database name from URL: %s", dbguard.RedactDSN(dsn))
			continue
		}
		if err := dbManager.AddConnection(name, dsn); err != nil {
			logger.Printf("Failed to auto-connect %s: %v", name, err)
		}
	}

	defer dbManager.CloseAll()

	logger.Println("Starting PostgreSQL MCP server...")
	if err := server.Serve(); err != nil {
		logger.Fatalf("Server error: %s", err)
	}
}
