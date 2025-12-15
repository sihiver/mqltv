package handlers

import (
	"net/http"
	"os"
	"strings"
)

func firstForwardedValue(v string) string {
	if v == "" {
		return ""
	}
	// X-Forwarded-* headers may contain a comma-separated list. We want the first.
	parts := strings.Split(v, ",")
	return strings.TrimSpace(parts[0])
}

// publicBaseURL returns the externally-reachable base URL for absolute links.
// Priority:
//  1. PUBLIC_BASE_URL (full URL, e.g. https://iptv.example.com)
//  2. X-Forwarded-Proto / X-Forwarded-Host (reverse proxy)
//  3. HOST env (host:port)
//  4. Request scheme (TLS?) + r.Host
//  5. localhost:8080
func publicBaseURL(r *http.Request) string {
	if v := strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}

	scheme := firstForwardedValue(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := firstForwardedValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(os.Getenv("HOST"))
	}
	if host == "" {
		host = r.Host
	}
	if host == "" {
		host = "localhost:8080"
	}

	return scheme + "://" + host
}
