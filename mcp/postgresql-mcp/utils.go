package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

var logger = log.New(os.Stderr, "", log.LstdFlags)

var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Response structures
type Response struct {
	OK       bool        `json:"ok"`
	Data     interface{} `json:"data,omitempty"`
	Error    string      `json:"error,omitempty"`
	RowCount *int        `json:"rowCount,omitempty"`
}

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
