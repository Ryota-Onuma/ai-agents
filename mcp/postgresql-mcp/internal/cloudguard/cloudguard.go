package cloudguard

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// CloudSQLProxyError represents an error when Cloud SQL Proxy connection is detected
type CloudSQLProxyError struct {
	DSN    string
	Reason string
}

func (e *CloudSQLProxyError) Error() string {
	return fmt.Sprintf("Cloud SQL Proxy connection rejected: %s (DSN: %s)", e.Reason, e.DSN)
}

// ValidateConnection checks if the DSN is a Cloud SQL Proxy connection and rejects it
func ValidateConnection(dsn string) error {
	// Check for Unix socket paths first (highest priority)
	if strings.Contains(dsn, "/cloudsql/") {
		return &CloudSQLProxyError{
			DSN:    dsn,
			Reason: "detected Cloud SQL Unix socket path (/cloudsql/…)",
		}
	}

	// Parse the DSN
	u, err := url.Parse(dsn)
	if err != nil {
		// If parsing fails, check for GCP instance patterns in the raw DSN
		gcpInstancePattern := regexp.MustCompile(`postgresql://[^@]*@[a-zA-Z0-9-]+:[a-zA-Z0-9-]+:[a-zA-Z0-9-]+`)
		if gcpInstancePattern.MatchString(dsn) {
			return &CloudSQLProxyError{
				DSN:    dsn,
				Reason: "detected Google Cloud SQL instance name pattern",
			}
		}
		// If we can't parse it and it's not a GCP pattern, let the original connection attempt handle it
		return nil
	}

	if hs := u.Query()["host"]; len(hs) > 0 {
		for _, h := range hs {
			if strings.HasPrefix(h, "/cloudsql/") {
				return &CloudSQLProxyError{
					DSN:    dsn,
					Reason: "detected Cloud SQL Unix socket path in ?host",
				}
			}
		}
	}

	// Check for Cloud SQL Proxy indicators
	if reason := detectCloudSQLProxy(u, dsn); reason != "" {
		return &CloudSQLProxyError{
			DSN:    dsn,
			Reason: reason,
		}
	}

	return nil
}

func detectCloudSQLProxy(u *url.URL, originalDSN string) string {
	// Check 1: Query parameters that suggest Cloud SQL
	if u.RawQuery != "" {
		params, err := url.ParseQuery(u.RawQuery)
		if err == nil {
			if _, hasInstance := params["instance"]; hasInstance {
				return "detected 'instance' parameter (common in Cloud SQL connections)"
			}
		}
	}

	// Check 2: Common Cloud SQL Proxy ports on localhost
	if u.Host != "" {
		host := u.Host
		// Remove port if present to check host alone
		if colonIdx := strings.LastIndex(host, ":"); colonIdx != -1 {
			port := host[colonIdx+1:]
			hostPart := host[:colonIdx]

			// Check for common Cloud SQL Proxy ports
			if isCloudSQLProxyPort(port) && (hostPart == "127.0.0.1" || hostPart == "localhost") {
				return fmt.Sprintf("detected localhost connection on Cloud SQL Proxy port %s", port)
			}
		}
	}

	// Check 3: Cloud SQL instance connection strings
	if strings.Contains(originalDSN, "cloudsql-proxy") ||
		strings.Contains(originalDSN, "cloud-sql-proxy") {
		return "detected Cloud SQL Proxy in connection string"
	}

	return ""
}

func isCloudSQLProxyPort(port string) bool {
	// Common Cloud SQL Proxy ports
	cloudSQLPorts := []string{
		"3307", // Most common
		"5433", // PostgreSQL alternative
		"1433", // SQL Server
		"3306", // MySQL (though less common for PostgreSQL)
		"5434", // Another PostgreSQL alternative
	}

	for _, cloudPort := range cloudSQLPorts {
		if port == cloudPort {
			return true
		}
	}

	return false
}
