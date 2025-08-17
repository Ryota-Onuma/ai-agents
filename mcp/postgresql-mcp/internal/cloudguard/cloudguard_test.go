package cloudguard

import (
	"strings"
	"testing"
)

func TestValidateConnection(t *testing.T) {
	tests := []struct {
		name        string
		dsn         string
		shouldError bool
		errorText   string
	}{
		{
			name:        "Valid regular PostgreSQL connection",
			dsn:         "postgresql://user:pass@localhost:5432/mydb",
			shouldError: false,
		},
		{
			name:        "Valid remote PostgreSQL connection",
			dsn:         "postgresql://user:pass@db.example.com:5432/mydb",
			shouldError: false,
		},
		{
			name:        "Cloud SQL Proxy on port 3307",
			dsn:         "postgresql://user:pass@127.0.0.1:3307/mydb",
			shouldError: true,
			errorText:   "detected localhost connection on Cloud SQL Proxy port 3307",
		},
		{
			name:        "Cloud SQL Proxy on port 5433",
			dsn:         "postgresql://user:pass@localhost:5433/mydb",
			shouldError: true,
			errorText:   "detected localhost connection on Cloud SQL Proxy port 5433",
		},
		{
			name:        "Cloud SQL Proxy string in DSN",
			dsn:         "postgresql://user:pass@cloudsql-proxy:5432/mydb",
			shouldError: true,
			errorText:   "detected Cloud SQL Proxy in connection string",
		},
		{
			name:        "Cloud SQL Unix socket path",
			dsn:         "postgresql://user:pass@/cloudsql/project:region:instance/mydb",
			shouldError: true,
			errorText:   "detected Cloud SQL Unix socket path",
		},
		{
			name:        "Google Cloud SQL instance pattern",
			dsn:         "postgresql://user:pass@my-project:us-central1:my-instance/mydb",
			shouldError: true,
			errorText:   "detected Google Cloud SQL instance name pattern",
		},
		{
			name:        "Cloud SQL with instance parameter",
			dsn:         "postgresql://user:pass@localhost:5432/mydb?instance=my-project:us-central1:my-instance",
			shouldError: true,
			errorText:   "detected 'instance' parameter",
		},
		{
			name:        "Cloud SQL with host parameter pointing to cloudsql",
			dsn:         "postgresql://user:pass@localhost:5432/mydb?host=/cloudsql/my-project:us-central1:my-instance",
			shouldError: true,
			errorText:   "detected Cloud SQL Unix socket path",
		},
		{
			name:        "Regular localhost connection on standard port",
			dsn:         "postgresql://user:pass@localhost:5432/mydb",
			shouldError: false,
		},
		{
			name:        "Invalid DSN format (should not error)",
			dsn:         "not-a-valid-dsn",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConnection(tt.dsn)
			
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for DSN %s, but got none", tt.dsn)
					return
				}
				
				// Check if it's the right type of error
				if cloudErr, ok := err.(*CloudSQLProxyError); ok {
					if !strings.Contains(cloudErr.Reason, tt.errorText) {
						t.Errorf("Expected error reason to contain '%s', got '%s'", tt.errorText, cloudErr.Reason)
					}
				} else {
					t.Errorf("Expected CloudSQLProxyError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for DSN %s, but got: %s", tt.dsn, err)
				}
			}
		})
	}
}

func TestDetectCloudSQLProxy(t *testing.T) {
	tests := []struct {
		name        string
		dsn         string
		expectError bool
	}{
		{"Standard PostgreSQL", "postgresql://user:pass@localhost:5432/db", false},
		{"Remote PostgreSQL", "postgresql://user:pass@db.example.com:5432/db", false},
		{"Cloud SQL Proxy port 3307", "postgresql://user:pass@127.0.0.1:3307/db", true},
		{"Cloud SQL Proxy port 5433", "postgresql://user:pass@localhost:5433/db", true},
		{"Cloud SQL in hostname", "postgresql://user:pass@cloudsql-proxy:5432/db", true},
		{"GCP instance pattern", "postgresql://user:pass@my-proj:us-cent1:my-inst/db", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConnection(tt.dsn)
			hasError := err != nil
			
			if hasError != tt.expectError {
				t.Errorf("DSN %s: expected error=%v, got error=%v (err: %v)", 
					tt.dsn, tt.expectError, hasError, err)
			}
		})
	}
}

func TestIsCloudSQLProxyPort(t *testing.T) {
	tests := []struct {
		port     string
		expected bool
	}{
		{"3307", true},
		{"5433", true},
		{"1433", true},
		{"3306", true},
		{"5434", true},
		{"5432", false},
		{"3000", false},
		{"8080", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.port, func(t *testing.T) {
			result := isCloudSQLProxyPort(tt.port)
			if result != tt.expected {
				t.Errorf("isCloudSQLProxyPort(%s) = %v, expected %v", tt.port, result, tt.expected)
			}
		})
	}
}