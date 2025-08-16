package main

import (
	"encoding/json"
	"testing"
)

func TestValidateIdentifier(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"table", true},
		{"_name", true},
		{"1table", false},
		{"invalid-name", false},
	}
	for _, tt := range tests {
		got, err := validateIdentifier(tt.input)
		if tt.valid {
			if err != nil {
				t.Fatalf("expected valid identifier %q, got error %v", tt.input, err)
			}
			expected := "\"" + tt.input + "\""
			if got != expected {
				t.Errorf("expected %s, got %s", expected, got)
			}
		} else {
			if err == nil {
				t.Errorf("expected error for %q", tt.input)
			}
		}
	}
}

func TestQIdent(t *testing.T) {
	got, err := qIdent("public.users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "\"public\".\"users\"" {
		t.Errorf("unexpected quoted ident: %s", got)
	}

	if _, err := qIdent("public.bad-name"); err == nil {
		t.Errorf("expected error for invalid identifier")
	}
}

func TestExtractDatabaseName(t *testing.T) {
	tests := []struct {
		dsn  string
		want string
	}{
		{"postgres://user:pass@localhost:5432/mydb?sslmode=disable", "mydb"},
		{"postgres://user@localhost", "postgres"},
		{"://bad-url", ""},
	}
	for _, tt := range tests {
		if got := extractDatabaseName(tt.dsn); got != tt.want {
			t.Errorf("extractDatabaseName(%q)=%q, want %q", tt.dsn, got, tt.want)
		}
	}
}

func TestResponses(t *testing.T) {
	rowCount := 1
	okResp := okResponse(map[string]int{"id": 5}, &rowCount)
	content := okResp["content"].([]map[string]interface{})[0]["text"].(string)
	var r Response
	if err := json.Unmarshal([]byte(content), &r); err != nil {
		t.Fatalf("failed to unmarshal ok response: %v", err)
	}
	if !r.OK || r.RowCount == nil || *r.RowCount != 1 {
		t.Errorf("unexpected ok response: %+v", r)
	}

	errResp := errResponse("failed")
	econtent := errResp["content"].([]map[string]interface{})[0]["text"].(string)
	var er Response
	if err := json.Unmarshal([]byte(econtent), &er); err != nil {
		t.Fatalf("failed to unmarshal err response: %v", err)
	}
	if er.OK || er.Error != "failed" {
		t.Errorf("unexpected err response: %+v", er)
	}
}
