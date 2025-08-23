package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func writeFakeGh(t *testing.T, content string, exitCode int) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gh")
	script := "#!/bin/sh\n" + content + "\nexit " + fmt.Sprintf("%d", exitCode) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake gh: %v", err)
	}
	return path
}

func TestGetAssignedPRLinks(t *testing.T) {
	fake := writeFakeGh(t, "echo https://example.com/pr1\necho\necho https://example.com/pr2", 0)
	t.Setenv("GH_COMMAND", fake)
	links, err := getAssignedPRLinks(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := []string{"https://example.com/pr1", "https://example.com/pr2"}
	if len(links) != len(want) {
		t.Fatalf("unexpected link count: %d", len(links))
	}
	for i, l := range links {
		if l != want[i] {
			t.Errorf("link %d = %s; want %s", i, l, want[i])
		}
	}
}

func TestGetAssignedPRLinksError(t *testing.T) {
	fake := writeFakeGh(t, "echo error 1>&2", 1)
	t.Setenv("GH_COMMAND", fake)
	if _, err := getAssignedPRLinks(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}
