package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Ryota-Onuma/ai-agents/mcp/postgresql-mcp/internal/cloudguard"
)

// Connection manager
type PostgreSQLManager struct {
	connections map[string]*sql.DB
	configs     map[string]string
}

func NewPostgreSQLManager() *PostgreSQLManager {
	return &PostgreSQLManager{
		connections: make(map[string]*sql.DB),
		configs:     make(map[string]string),
	}
}

func (m *PostgreSQLManager) AddConnection(name, dsn string) error {
	// Validate connection against Cloud SQL Proxy
	if err := cloudguard.ValidateConnection(dsn); err != nil {
		logger.Printf("Connection rejected for %s: %s", name, err)
		return fmt.Errorf("connection rejected: %w", err)
	}

	// Close existing connection if re-adding
	if db, exists := m.connections[name]; exists {
		db.Close()
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	m.connections[name] = db
	m.configs[name] = dsn
	logger.Printf("Connected to database: %s", name)
	return nil
}

func (m *PostgreSQLManager) HasConnection(name string) bool {
	if name == "" {
		return len(m.connections) > 0
	}
	_, exists := m.connections[name]
	return exists
}

func (m *PostgreSQLManager) ListConnections() []string {
	var names []string
	for name := range m.connections {
		names = append(names, name)
	}
	return names
}

func (m *PostgreSQLManager) GetConnection(name string) *sql.DB {
	if name != "" {
		return m.connections[name]
	}
	// Return first connection if unspecified
	for _, db := range m.connections {
		return db
	}
	return nil
}

func (m *PostgreSQLManager) Disconnect(name string) bool {
	if db, exists := m.connections[name]; exists {
		db.Close()
		delete(m.connections, name)
		delete(m.configs, name)
		logger.Printf("Closed connection: %s", name)
		return true
	}
	return false
}

func (m *PostgreSQLManager) CloseAll() {
	for name, db := range m.connections {
		db.Close()
		logger.Printf("Closed connection: %s", name)
	}
	m.connections = make(map[string]*sql.DB)
	m.configs = make(map[string]string)
}

var dbManager = NewPostgreSQLManager()

// extractDatabaseName extracts database name from PostgreSQL connection URL
func extractDatabaseName(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		logger.Printf("Failed to parse URL %s: %s", dsn, err)
		return ""
	}

	// Remove leading slash from path to get database name
	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		return "postgres" // default database name
	}
	return dbName
}

// Helper function to execute with timeout
func execWithTimeout(db *sql.DB, timeoutMs *int, query string, args ...interface{}) (*sql.Rows, error) {
	ctx := context.Background()
	if timeoutMs != nil && *timeoutMs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*timeoutMs)*time.Millisecond)
		defer cancel()
	}
	return db.QueryContext(ctx, query, args...)
}

func execWithTimeoutSingle(db *sql.DB, timeoutMs *int, query string, args ...interface{}) error {
	ctx := context.Background()
	if timeoutMs != nil && *timeoutMs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*timeoutMs)*time.Millisecond)
		defer cancel()
	}
	_, err := db.ExecContext(ctx, query, args...)
	return err
}
