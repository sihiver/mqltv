package handlers

import (
	"net"
	"net/http"
	"os"
	"strconv"
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

func isLocalhostHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "" {
		return false
	}
	// Strip port if present (best-effort)
	if strings.HasPrefix(h, "[") {
		// IPv6 in brackets; keep as-is
	} else if strings.Count(h, ":") == 1 {
		if hp, _, err := net.SplitHostPort(h); err == nil {
			h = hp
		}
	}

	return h == "localhost" || h == "127.0.0.1" || h == "::1" || h == "[::1]"
}

func isPrivateIPv4(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	// 10.0.0.0/8
	if ip4[0] == 10 {
		return true
	}
	// 172.16.0.0/12
	if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
		return true
	}
	// 192.168.0.0/16
	if ip4[0] == 192 && ip4[1] == 168 {
		return true
	}
	return false
}

func detectLANIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	var firstNonLoopback string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			if ip4.IsLoopback() {
				continue
			}
			if firstNonLoopback == "" {
				firstNonLoopback = ip4.String()
			}
			if isPrivateIPv4(ip4) {
				return ip4.String()
			}
		}
	}

	return firstNonLoopback
}

func defaultPortFromRequest(r *http.Request) string {
	if r != nil && r.Host != "" {
		if _, port, err := net.SplitHostPort(r.Host); err == nil {
			return port
		}
	}
	if p := strings.TrimSpace(os.Getenv("PORT")); p != "" {
		if _, err := strconv.Atoi(p); err == nil {
			return p
		}
	}
	return "8080"
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
	// If we're being called via a dev proxy / local address, the Host header may be "localhost"
	// which would cause generated absolute URLs to be unusable for other devices.
	// In that case, best-effort detect a LAN IPv4 and use it.
	if isLocalhostHost(host) {
		if ip := detectLANIPv4(); ip != "" {
			port := defaultPortFromRequest(r)
			host = ip + ":" + port
		}
	}
	if host == "" {
		host = "localhost:8080"
	}

	return scheme + "://" + host
}
