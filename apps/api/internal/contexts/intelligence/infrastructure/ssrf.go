package infrastructure

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateEndpoint guards against SSRF when the app dials provider/connector
// URLs. External endpoints must be https. Loopback endpoints (Ollama, local
// gateways) are allowed only when allowLocal is true; the dial is additionally
// pinned to the loopback interface by the caller where possible.
func ValidateEndpoint(raw string, allowLocal bool) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid endpoint: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("endpoint must use http(s), got %q", u.Scheme)
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("endpoint host is required")
	}

	ip := net.ParseIP(host)
	// Always reject metadata/private/link-local/unspecified targets: an
	// allowlist of "local providers" must not become a proxy into the LAN or
	// cloud metadata endpoints.
	if ip != nil && (ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast()) {
		return fmt.Errorf("endpoint %q targets a private or link-local address", host)
	}

	if isLocalHost(host) {
		if !allowLocal {
			return fmt.Errorf("loopback endpoint requires explicit local-provider mode")
		}
		return nil
	}
	if u.Scheme != "https" {
		return fmt.Errorf("non-loopback endpoints must use https")
	}
	return nil
}

// isLocalHost reports whether the host is explicitly local (localhost or a
// loopback IP).
func isLocalHost(host string) bool {
	if host == "localhost" {
		return true
	}
	if strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}
