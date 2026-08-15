package infrastructure

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ValidateEndpoint guards against SSRF when the app dials provider/connector
// URLs. External endpoints must be https. Loopback endpoints (Ollama, local
// gateways) are allowed only when allowLocal is true. The host is additionally
// resolved and any private/link-local/loopback answer is rejected unless the
// target is explicitly local.
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

	if err := validateHost(host, allowLocal); err != nil {
		return err
	}
	if !isLocalHost(host) && u.Scheme != "https" {
		return fmt.Errorf("non-loopback endpoints must use https")
	}
	return nil
}

// validateHost applies the address policy to a hostname, resolving DNS so a
// host that only resolves to private addresses is rejected.
func validateHost(host string, allowLocal bool) error {
	if ip := net.ParseIP(host); ip != nil {
		return validateIP(ip, host, allowLocal)
	}

	// Hostname: check the resolved addresses (metadata endpoints and LAN
	// rebinding land here).
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		// Unresolvable now — let the dial fail naturally; policy still applies
		// to the literal host string below.
		addrs = nil
	}
	seen := false
	for _, a := range addrs {
		seen = true
		if err := validateIP(a.IP, host, allowLocal); err != nil {
			return err
		}
	}
	_ = seen
	return nil
}

func validateIP(ip net.IP, host string, allowLocal bool) error {
	switch {
	case ip.IsLoopback():
		if !allowLocal {
			return fmt.Errorf("loopback endpoint %q requires explicit local-provider mode", host)
		}
		return nil
	case ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast():
		return fmt.Errorf("endpoint %q targets a private or link-local address", host)
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
	return ip != nil && ip.IsLoopback()
}

// CheckRedirectFor returns an http.Client.CheckRedirect policy that re-applies
// the same endpoint policy to every redirect target, blocking hops that would
// escape to private/loopback/metadata addresses.
func CheckRedirectFor(allowLocal bool) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("stopped after 5 redirects")
		}
		if err := ValidateEndpoint(req.URL.String(), allowLocal); err != nil {
			return fmt.Errorf("redirect blocked: %w", err)
		}
		return nil
	}
}
