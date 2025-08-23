package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func ghCmd() string {
	if c := os.Getenv("GH_COMMAND"); c != "" {
		return c
	}
	return "gh"
}

func getAssignedPRLinks(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, ghCmd(), "pr", "list", "--assignee", "@me", "--json", "url", "--jq", ".[].url")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh command failed: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var links []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			links = append(links, line)
		}
	}
	return links, nil
}

func main() {
	log.SetFlags(0)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	links, err := getAssignedPRLinks(ctx)
	if err != nil {
		log.Fatalf(`{"level":"error","msg":"%v"}`, err)
	}
	log.Printf(`{"level":"info","msg":"found pull requests","count":%d}`, len(links))
	for _, l := range links {
		fmt.Println(l)
	}
}
