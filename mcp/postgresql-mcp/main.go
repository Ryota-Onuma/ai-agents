package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// Logging
var logger = log.New(os.Stderr, "", log.LstdFlags)

// Identifier validation
var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// validateIdentifier validates one identifier part and returns the double-quoted identifier
func validateIdentifier(token string) (string, error) {
	if !identRe.MatchString(token) {
		return "", fmt.Errorf("invalid identifier: %q", token)
	}
	return fmt.Sprintf(`"%s"`, token), nil
}

// qIdent quotes identifier possibly with schema: "schema"."table" or "column"
func qIdent(ident string) (string, error) {
	parts := strings.Split(ident, ".")
	var quotedParts []string
	for _, part := range parts {
		quoted, err := validateIdentifier(part)
		if err != nil {
			return "", err
		}
		quotedParts = append(quotedParts, quoted)
	}
	return strings.Join(quotedParts, "."), nil
}

// Response structures
type Response struct {
	OK       bool        `json:"ok"`
	Data     interface{} `json:"data,omitempty"`
	Error    string      `json:"error,omitempty"`
	RowCount *int        `json:"rowCount,omitempty"`
}

// MCP Server implementation
type MCPServer struct {
	name    string
	version string
	tools   map[string]Tool
}

type Tool struct {
	name        string
	description string
	schema      map[string]interface{}
	handler     func(map[string]interface{}) map[string]interface{}
}

func NewMCPServer(name, version string) *MCPServer {
	return &MCPServer{
		name:    name,
		version: version,
		tools:   make(map[string]Tool),
	}
}

func (s *MCPServer) AddTool(name, description string, schema map[string]interface{}, handler func(map[string]interface{}) map[string]interface{}) {
	s.tools[name] = Tool{
		name:        name,
		description: description,
		schema:      schema,
		handler:     handler,
	}
}

func (s *MCPServer) Serve() error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var request map[string]interface{}
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			s.sendError(-32700, "Parse error", nil)
			continue
		}

		s.handleRequest(request)
	}
	return scanner.Err()
}

func (s *MCPServer) handleRequest(request map[string]interface{}) {
	method, ok := request["method"].(string)
	if !ok {
		s.sendError(-32600, "Invalid Request", request["id"])
		return
	}

	switch method {
	case "initialize":
		s.handleInitialize(request)
	case "tools/list":
		s.handleToolsList(request)
	case "tools/call":
		s.handleToolsCall(request)
	default:
		s.sendError(-32601, "Method not found", request["id"])
	}
}

func (s *MCPServer) handleInitialize(request map[string]interface{}) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      request["id"],
		"result": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    s.name,
				"version": s.version,
			},
		},
	}
	s.sendResponse(response)
}

func (s *MCPServer) handleToolsList(request map[string]interface{}) {
	var tools []map[string]interface{}
	for _, tool := range s.tools {
		tools = append(tools, map[string]interface{}{
			"name":        tool.name,
			"description": tool.description,
			"inputSchema": tool.schema,
		})
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      request["id"],
		"result": map[string]interface{}{
			"tools": tools,
		},
	}
	s.sendResponse(response)
}

func (s *MCPServer) handleToolsCall(request map[string]interface{}) {
	params, ok := request["params"].(map[string]interface{})
	if !ok {
		s.sendError(-32602, "Invalid params", request["id"])
		return
	}

	name, ok := params["name"].(string)
	if !ok {
		s.sendError(-32602, "Tool name required", request["id"])
		return
	}

	tool, exists := s.tools[name]
	if !exists {
		s.sendError(-32602, "Tool not found", request["id"])
		return
	}

	args, _ := params["arguments"].(map[string]interface{})
	result := tool.handler(args)

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      request["id"],
		"result":  result,
	}
	s.sendResponse(response)
}

func (s *MCPServer) sendResponse(response map[string]interface{}) {
	data, err := json.Marshal(response)
	if err != nil {
		logger.Printf("Failed to marshal response: %s", err)
		fmt.Println(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"}}`)
		return
	}
	fmt.Println(string(data))
}

func (s *MCPServer) sendError(code int, message string, id interface{}) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	s.sendResponse(response)
}

func okResponse(data interface{}, rowCount *int) map[string]interface{} {
	resp := Response{OK: true}
	if data != nil {
		resp.Data = data
	}
	if rowCount != nil {
		resp.RowCount = rowCount
	}
	b, err := json.Marshal(resp)
	if err != nil {
		logger.Printf("Failed to marshal response: %s", err)
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": fmt.Sprintf(`{"ok":false,"error":"Failed to marshal response: %s"}`, err)},
			},
		}
	}
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": string(b)},
		},
	}
}

func errResponse(msg string) map[string]interface{} {
	resp := Response{OK: false, Error: msg}
	b, err := json.Marshal(resp)
	if err != nil {
		logger.Printf("Failed to marshal error response: %s", err)
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": fmt.Sprintf(`{"ok":false,"error":"Marshal error: %s"}`, err)},
			},
		}
	}
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": string(b)},
		},
	}
}

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

// setupDatabaseConnections sets up database connections from environment variables
func setupDatabaseConnections() {
	// Support for POSTGRESQL_URLS environment variable (comma-separated)
	if urlsEnv := os.Getenv("POSTGRESQL_URLS"); urlsEnv != "" {
		urls := strings.Split(urlsEnv, ",")
		for _, url := range urls {
			url = strings.TrimSpace(url)
			if url != "" {
				name := extractDatabaseName(url)
				if name == "" {
					logger.Printf("Failed to extract database name from URL: %s", url)
					continue
				}
				if err := dbManager.AddConnection(name, url); err != nil {
					logger.Printf("Failed to auto-connect %s: %s", name, err)
				}
			}
		}
		return
	}
	logger.Println("No DB URLs are passed..")
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

// Tool handlers
func connectHandler(args map[string]interface{}) map[string]interface{} {
	name, ok := args["name"].(string)
	if !ok {
		return errResponse("name is required")
	}

	dsn, ok := args["connection_string"].(string)
	if !ok {
		return errResponse("connection_string is required")
	}

	if err := dbManager.AddConnection(name, dsn); err != nil {
		return errResponse(fmt.Sprintf("Failed to connect: %s", err))
	}

	return okResponse(map[string]string{"name": name}, nil)
}

func disconnectHandler(args map[string]interface{}) map[string]interface{} {
	name, ok := args["name"].(string)
	if !ok {
		return errResponse("name is required")
	}

	if dbManager.Disconnect(name) {
		return okResponse(map[string]string{"name": name}, nil)
	}
	return errResponse(fmt.Sprintf("No such connection: %s", name))
}

func listConnectionsHandler(args map[string]interface{}) map[string]interface{} {
	return okResponse(dbManager.ListConnections(), nil)
}

func queryHandler(args map[string]interface{}) map[string]interface{} {
	sqlQuery, ok := args["sql"].(string)
	if !ok {
		return errResponse("sql is required")
	}

	var params []interface{}
	if p, exists := args["params"]; exists {
		if paramSlice, ok := p.([]interface{}); ok {
			params = paramSlice
		}
	}

	var database string
	if d, exists := args["database"]; exists {
		if dbStr, ok := d.(string); ok {
			database = dbStr
		}
	}

	var limit *int
	if l, exists := args["limit"]; exists {
		if limitFloat, ok := l.(float64); ok {
			limitInt := int(limitFloat)
			limit = &limitInt
		}
	}

	var timeoutMs *int
	if t, exists := args["statement_timeout_ms"]; exists {
		if timeoutFloat, ok := t.(float64); ok {
			timeoutInt := int(timeoutFloat)
			timeoutMs = &timeoutInt
		}
	}

	// Wrap with LIMIT if requested
	if limit != nil && *limit > 0 {
		sqlQuery = fmt.Sprintf("SELECT * FROM (%s) AS __q LIMIT %d", sqlQuery, *limit)
	}

	if !dbManager.HasConnection(database) {
		return errResponse("No database connection available")
	}

	db := dbManager.GetConnection(database)
	if db == nil {
		return errResponse("No database connection available")
	}

	rows, err := execWithTimeout(db, timeoutMs, sqlQuery, params...)
	if err != nil {
		return errResponse(fmt.Sprintf("Query failed: %s", err))
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return errResponse(fmt.Sprintf("Failed to get columns: %s", err))
	}

	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return errResponse(fmt.Sprintf("Failed to scan row: %s", err))
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		result = append(result, row)
	}

	rowCount := len(result)
	return okResponse(result, &rowCount)
}

func insertHandler(args map[string]interface{}) map[string]interface{} {
	table, ok := args["table"].(string)
	if !ok {
		return errResponse("table is required")
	}

	data, ok := args["data"].(map[string]interface{})
	if !ok || len(data) == 0 {
		return errResponse("data must be a non-empty object")
	}

	var database string
	if d, exists := args["database"]; exists {
		if dbStr, ok := d.(string); ok {
			database = dbStr
		}
	}

	returning := true
	if r, exists := args["returning"]; exists {
		if retBool, ok := r.(bool); ok {
			returning = retBool
		}
	}

	var timeoutMs *int
	if t, exists := args["statement_timeout_ms"]; exists {
		if timeoutFloat, ok := t.(float64); ok {
			timeoutInt := int(timeoutFloat)
			timeoutMs = &timeoutInt
		}
	}

	if !dbManager.HasConnection(database) {
		return errResponse("No database connection available")
	}

	db := dbManager.GetConnection(database)
	if db == nil {
		return errResponse("No database connection available")
	}

	// Build INSERT query
	quotedTable, err := qIdent(table)
	if err != nil {
		return errResponse(err.Error())
	}

	var columns []string
	var placeholders []string
	var values []interface{}
	i := 1
	for col, val := range data {
		quotedCol, err := qIdent(col)
		if err != nil {
			return errResponse(err.Error())
		}
		columns = append(columns, quotedCol)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, val)
		i++
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quotedTable,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	if returning {
		query += " RETURNING *"
		rows, err := execWithTimeout(db, timeoutMs, query, values...)
		if err != nil {
			return errResponse(fmt.Sprintf("Insert failed: %s", err))
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return errResponse(fmt.Sprintf("Failed to get columns: %s", err))
		}

		if rows.Next() {
			rowValues := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range rowValues {
				valuePtrs[i] = &rowValues[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return errResponse(fmt.Sprintf("Failed to scan row: %s", err))
			}

			row := make(map[string]interface{})
			for i, col := range columns {
				val := rowValues[i]
				if b, ok := val.([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
			}
			rowCount := 1
			return okResponse(row, &rowCount)
		}
	} else {
		if err := execWithTimeoutSingle(db, timeoutMs, query, values...); err != nil {
			return errResponse(fmt.Sprintf("Insert failed: %s", err))
		}
	}

	rowCount := 1
	return okResponse(nil, &rowCount)
}

func updateHandler(args map[string]interface{}) map[string]interface{} {
	table, ok := args["table"].(string)
	if !ok {
		return errResponse("table is required")
	}

	data, ok := args["data"].(map[string]interface{})
	if !ok || len(data) == 0 {
		return errResponse("data must be a non-empty object")
	}

	where, ok := args["where"].(map[string]interface{})
	if !ok || len(where) == 0 {
		return errResponse("where must be a non-empty object")
	}

	var database string
	if d, exists := args["database"]; exists {
		if dbStr, ok := d.(string); ok {
			database = dbStr
		}
	}

	returning := false
	if r, exists := args["returning"]; exists {
		if retBool, ok := r.(bool); ok {
			returning = retBool
		}
	}

	var timeoutMs *int
	if t, exists := args["statement_timeout_ms"]; exists {
		if timeoutFloat, ok := t.(float64); ok {
			timeoutInt := int(timeoutFloat)
			timeoutMs = &timeoutInt
		}
	}

	if !dbManager.HasConnection(database) {
		return errResponse("No database connection available")
	}

	db := dbManager.GetConnection(database)
	if db == nil {
		return errResponse("No database connection available")
	}

	// Build UPDATE query
	quotedTable, err := qIdent(table)
	if err != nil {
		return errResponse(err.Error())
	}

	var setClauses []string
	var whereClauses []string
	var values []interface{}
	i := 1

	// SET clause
	for col, val := range data {
		quotedCol, err := qIdent(col)
		if err != nil {
			return errResponse(err.Error())
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quotedCol, i))
		values = append(values, val)
		i++
	}

	// WHERE clause
	for col, val := range where {
		quotedCol, err := qIdent(col)
		if err != nil {
			return errResponse(err.Error())
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", quotedCol, i))
		values = append(values, val)
		i++
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		quotedTable,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "))

	if returning {
		query += " RETURNING *"
		rows, err := execWithTimeout(db, timeoutMs, query, values...)
		if err != nil {
			return errResponse(fmt.Sprintf("Update failed: %s", err))
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return errResponse(fmt.Sprintf("Failed to get columns: %s", err))
		}

		var result []map[string]interface{}
		for rows.Next() {
			rowValues := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range rowValues {
				valuePtrs[i] = &rowValues[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return errResponse(fmt.Sprintf("Failed to scan row: %s", err))
			}

			row := make(map[string]interface{})
			for i, col := range columns {
				val := rowValues[i]
				if b, ok := val.([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
			}
			result = append(result, row)
		}

		rowCount := len(result)
		return okResponse(result, &rowCount)
	} else {
		result, err := db.Exec(query, values...)
		if err != nil {
			return errResponse(fmt.Sprintf("Update failed: %s", err))
		}
		rowsAffected, _ := result.RowsAffected()
		count := int(rowsAffected)
		return okResponse(nil, &count)
	}
}

func deleteHandler(args map[string]interface{}) map[string]interface{} {
	table, ok := args["table"].(string)
	if !ok {
		return errResponse("table is required")
	}

	where, ok := args["where"].(map[string]interface{})
	if !ok || len(where) == 0 {
		return errResponse("where must be a non-empty object")
	}

	var database string
	if d, exists := args["database"]; exists {
		if dbStr, ok := d.(string); ok {
			database = dbStr
		}
	}

	returning := false
	if r, exists := args["returning"]; exists {
		if retBool, ok := r.(bool); ok {
			returning = retBool
		}
	}

	var timeoutMs *int
	if t, exists := args["statement_timeout_ms"]; exists {
		if timeoutFloat, ok := t.(float64); ok {
			timeoutInt := int(timeoutFloat)
			timeoutMs = &timeoutInt
		}
	}

	if !dbManager.HasConnection(database) {
		return errResponse("No database connection available")
	}

	db := dbManager.GetConnection(database)
	if db == nil {
		return errResponse("No database connection available")
	}

	// Build DELETE query
	quotedTable, err := qIdent(table)
	if err != nil {
		return errResponse(err.Error())
	}

	var whereClauses []string
	var values []interface{}
	i := 1

	for col, val := range where {
		quotedCol, err := qIdent(col)
		if err != nil {
			return errResponse(err.Error())
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", quotedCol, i))
		values = append(values, val)
		i++
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s",
		quotedTable,
		strings.Join(whereClauses, " AND "))

	if returning {
		query += " RETURNING *"
		rows, err := execWithTimeout(db, timeoutMs, query, values...)
		if err != nil {
			return errResponse(fmt.Sprintf("Delete failed: %s", err))
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return errResponse(fmt.Sprintf("Failed to get columns: %s", err))
		}

		var result []map[string]interface{}
		for rows.Next() {
			rowValues := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range rowValues {
				valuePtrs[i] = &rowValues[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return errResponse(fmt.Sprintf("Failed to scan row: %s", err))
			}

			row := make(map[string]interface{})
			for i, col := range columns {
				val := rowValues[i]
				if b, ok := val.([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
			}
			result = append(result, row)
		}

		rowCount := len(result)
		return okResponse(result, &rowCount)
	} else {
		result, err := db.Exec(query, values...)
		if err != nil {
			return errResponse(fmt.Sprintf("Delete failed: %s", err))
		}
		rowsAffected, _ := result.RowsAffected()
		count := int(rowsAffected)
		return okResponse(nil, &count)
	}
}

func listSchemasHandler(args map[string]interface{}) map[string]interface{} {
	var database string
	if d, exists := args["database"]; exists {
		if dbStr, ok := d.(string); ok {
			database = dbStr
		}
	}

	if !dbManager.HasConnection(database) {
		return errResponse("No database connection available")
	}

	db := dbManager.GetConnection(database)
	if db == nil {
		return errResponse("No database connection available")
	}

	query := `
		SELECT nspname AS schema
		FROM pg_namespace
		WHERE nspname NOT LIKE 'pg_%' AND nspname <> 'information_schema'
		ORDER BY nspname
	`

	rows, err := db.Query(query)
	if err != nil {
		return errResponse(fmt.Sprintf("Query failed: %s", err))
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return errResponse(fmt.Sprintf("Failed to scan row: %s", err))
		}
		schemas = append(schemas, schema)
	}

	return okResponse(schemas, nil)
}

func listTablesHandler(args map[string]interface{}) map[string]interface{} {
	var database string
	if d, exists := args["database"]; exists {
		if dbStr, ok := d.(string); ok {
			database = dbStr
		}
	}

	schema := "public"
	if s, exists := args["schema"]; exists {
		if schemaStr, ok := s.(string); ok {
			schema = schemaStr
		}
	}

	if !dbManager.HasConnection(database) {
		return errResponse("No database connection available")
	}

	db := dbManager.GetConnection(database)
	if db == nil {
		return errResponse("No database connection available")
	}

	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1
		ORDER BY table_name
	`

	rows, err := db.Query(query, schema)
	if err != nil {
		return errResponse(fmt.Sprintf("Query failed: %s", err))
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return errResponse(fmt.Sprintf("Failed to scan row: %s", err))
		}
		tables = append(tables, table)
	}

	return okResponse(tables, nil)
}

func describeTableHandler(args map[string]interface{}) map[string]interface{} {
	var database string
	if d, exists := args["database"]; exists {
		if dbStr, ok := d.(string); ok {
			database = dbStr
		}
	}

	table, ok := args["table"].(string)
	if !ok {
		return errResponse("table is required")
	}

	if !dbManager.HasConnection(database) {
		return errResponse("No database connection available")
	}

	db := dbManager.GetConnection(database)
	if db == nil {
		return errResponse("No database connection available")
	}

	// Split schema.table if provided
	parts := strings.Split(table, ".")
	var schema, tableName string
	if len(parts) == 2 {
		schema = parts[0]
		tableName = parts[1]
	} else {
		schema = "public"
		tableName = parts[0]
	}

	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`

	rows, err := db.Query(query, schema, tableName)
	if err != nil {
		return errResponse(fmt.Sprintf("Query failed: %s", err))
	}
	defer rows.Close()

	var columns []map[string]interface{}
	for rows.Next() {
		var columnName, dataType, isNullable string
		var columnDefault sql.NullString

		if err := rows.Scan(&columnName, &dataType, &isNullable, &columnDefault); err != nil {
			return errResponse(fmt.Sprintf("Failed to scan row: %s", err))
		}

		column := map[string]interface{}{
			"column_name":    columnName,
			"data_type":      dataType,
			"is_nullable":    isNullable,
			"column_default": nil,
		}

		if columnDefault.Valid {
			column["column_default"] = columnDefault.String
		}

		columns = append(columns, column)
	}

	return okResponse(columns, nil)
}

func main() {
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

	// Auto-connect via environment variables
	setupDatabaseConnections()

	defer dbManager.CloseAll()

	logger.Println("Starting PostgreSQL MCP server...")
	if err := server.Serve(); err != nil {
		logger.Fatalf("Server error: %s", err)
	}
}
