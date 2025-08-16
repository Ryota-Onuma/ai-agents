package main

import (
	"database/sql"
	"fmt"
	"strings"
)

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
