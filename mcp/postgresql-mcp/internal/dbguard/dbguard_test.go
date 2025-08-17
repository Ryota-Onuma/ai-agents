package dbguard

import (
	"testing"
)

func TestSplitCommaRespectingEscape(t *testing.T) {
	got := splitCommaRespectingEscape("a\\,b,c")
	if len(got) != 2 || got[0] != "a,b" || got[1] != "c" {
		t.Fatalf("unexpected split result: %#v", got)
	}
}

func TestLoadPostgresURLsFromArgs(t *testing.T) {
	urlString := "postgres://u:p@localhost/db1,postgres://u:p@localhost/db2"
	urls, err := LoadPostgresURLsFromArgs(urlString)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 2 {
		t.Fatalf("expected 2 urls, got %d", len(urls))
	}
	
	// Test empty string
	_, err = LoadPostgresURLsFromArgs("")
	if err == nil {
		t.Fatalf("expected error for empty string")
	}
}


func TestEnforceLocalForURLsAllow(t *testing.T) {
	dsns := []string{"postgresql://u:p@localhost:5432/db?sslmode=disable", "postgresql://u:p@[::1]:5432/db"}
	got, err := EnforceLocalForURLs(dsns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(dsns) {
		t.Fatalf("expected %d dsns, got %d", len(dsns), len(got))
	}
}

func TestEnforceLocalForURLsReject(t *testing.T) {
	dsns := []string{"postgresql://u:p@10.0.0.5:5432/db"}
	if _, err := EnforceLocalForURLs(dsns); err == nil {
		t.Fatalf("expected error for remote ip")
	}
}

func TestEnforceLocalForURLsRejectMixed(t *testing.T) {
	dsns := []string{"postgresql://u:p@localhost,10.0.0.5:5432/db"}
	if _, err := EnforceLocalForURLs(dsns); err == nil {
		t.Fatalf("expected error for mixed hosts")
	}
}

func TestRedactDSN(t *testing.T) {
	dsn := "postgresql://user:secret@localhost/db"
	redacted := RedactDSN(dsn)
	if redacted != "postgresql://user:***@localhost/db" {
		t.Fatalf("password not redacted: %s", redacted)
	}
}
