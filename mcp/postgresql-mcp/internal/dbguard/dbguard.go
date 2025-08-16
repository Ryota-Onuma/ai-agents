package dbguard

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var (
	allowedSocketRoots = []string{"/var/run/postgresql", "/tmp", "/var/pgsql_socket"}
	errRemote          = errors.New("remote database is not allowed (local only)")
)

// LoadPostgresURLsFromEnv loads comma-separated PostgreSQL URLs from environment variable
func LoadPostgresURLsFromEnv() ([]string, error) {
	s := strings.TrimSpace(os.Getenv("POSTGRESQL_URLS"))
	if s == "" {
		return nil, errors.New("POSTGRESQL_URLS not set")
	}
	items := splitCommaRespectingEscape(s)
	out := make([]string, 0, len(items))
	for _, it := range items {
		it = strings.TrimSpace(strings.ReplaceAll(it, `\,`, `,`))
		if it != "" {
			out = append(out, it)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("POSTGRESQL_URLS contains no usable URL")
	}
	return out, nil
}

// EnforceLocalForURLs validates URLs are local and returns the original DSNs
func EnforceLocalForURLs(urls []string) ([]string, error) {
	out := make([]string, 0, len(urls))
	for _, dsn := range urls {
		u, err := url.Parse(dsn)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %q: %w", RedactDSN(dsn), err)
		}
		if err := validateLocalURL(u); err != nil {
			return nil, fmt.Errorf("%w: %s", err, RedactDSN(dsn))
		}
		out = append(out, dsn)
	}
	return out, nil
}

func validateLocalURL(u *url.URL) error {
	if hosts := u.Query()["host"]; len(hosts) > 0 {
		for _, h := range hosts {
			if !strings.HasPrefix(h, "/") || !isUnderAllowedSocketRoots(h) {
				return fmt.Errorf("%w: unix socket outside allowed roots: %q", errRemote, h)
			}
		}
		return nil
	}

	rawHost := u.Host
	if rawHost == "" {
		return fmt.Errorf("missing host")
	}
	for _, hp := range splitHostListPreservingIPv6(rawHost) {
		h := hp
		if strings.Contains(hp, ":") {
			if hh, _, err := net.SplitHostPort(hp); err == nil {
				h = hh
			} else {
				return fmt.Errorf("invalid host:port: %q", hp)
			}
		}
		h = strings.Trim(h, "[]")

		if strings.HasPrefix(h, "/") {
			if !isUnderAllowedSocketRoots(h) {
				return fmt.Errorf("%w: unix socket outside allowed roots: %q", errRemote, h)
			}
			continue
		}

		if ip := net.ParseIP(h); ip != nil {
			if !ip.IsLoopback() {
				return fmt.Errorf("%w: non-loopback ip: %s", errRemote, ip.String())
			}
			continue
		}

		ips, err := net.LookupIP(h)
		if err != nil || len(ips) == 0 {
			return fmt.Errorf("host resolve failed: %q", h)
		}
		for _, ip := range ips {
			if !ip.IsLoopback() {
				return fmt.Errorf("%w: host resolves to non-loopback: %s -> %s", errRemote, h, ip.String())
			}
		}
	}
	return nil
}

func isUnderAllowedSocketRoots(sock string) bool {
	real, err := filepath.EvalSymlinks(sock)
	if err != nil {
		return false
	}
	abs, err := filepath.Abs(real)
	if err != nil {
		return false
	}
	for _, root := range allowedSocketRoots {
		rabs, _ := filepath.Abs(root)
		if within(abs, rabs) {
			return true
		}
	}
	return false
}

func within(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, "..")
}

func splitHostListPreservingIPv6(raw string) []string {
	out := []string{}
	var b strings.Builder
	br := 0
	for _, r := range raw {
		switch r {
		case '[':
			br++
		case ']':
			if br > 0 {
				br--
			}
		case ',':
			if br == 0 {
				out = append(out, b.String())
				b.Reset()
				continue
			}
		}
		b.WriteRune(r)
	}
	out = append(out, b.String())
	return out
}

func splitCommaRespectingEscape(s string) []string {
	out := []string{}
	var b strings.Builder
	esc := false
	for _, r := range s {
		if esc {
			b.WriteRune(r)
			esc = false
			continue
		}
		if r == '\\' {
			esc = true
			continue
		}
		if r == ',' {
			out = append(out, b.String())
			b.Reset()
			continue
		}
		b.WriteRune(r)
	}
	out = append(out, b.String())
	return out
}

func RedactDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil || u.User == nil {
		return dsn
	}
	if _, has := u.User.Password(); has {
		u.User = url.UserPassword(u.User.Username(), "***")
		return strings.Replace(u.String(), "%2A%2A%2A", "***", 1)
	}
	return u.String()
}
