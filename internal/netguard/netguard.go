// Package netguard locks all outbound network access to an allowlist.
//
// In this build Crush may only reach:
//   - the custom model endpoint baked in at build time (config.CustomBaseURL)
//   - an optional analytics endpoint baked in at build time (config.AnalyticsURL)
//   - loopback (localhost / 127.0.0.1 / ::1) — the local API/LSP/MCP servers
//
// Everything else (catwalk, hyper, data.charm.land, github, google, openai, …) is
// refused at dial time. Install replaces http.DefaultTransport (and DefaultClient) with a
// guarded transport, so every HTTP client that uses the defaults — which is essentially
// all of Crush's and its libraries' clients — is covered. Fail closed: unknown hosts are
// blocked, not allowed.
package netguard

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/charmbracelet/crush/internal/config"
)

var once sync.Once

func hostOf(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func allowedHosts() map[string]struct{} {
	hosts := map[string]struct{}{
		"localhost": {},
		"127.0.0.1": {},
		"::1":       {},
		"0.0.0.0":   {},
	}
	if h := hostOf(config.CustomBaseURL()); h != "" {
		hosts[h] = struct{}{}
	}
	if h := hostOf(config.AnalyticsURL); h != "" {
		hosts[h] = struct{}{}
	}
	return hosts
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Install replaces the default HTTP transport with one that only dials allowed hosts.
// Call this once, as early as possible in main(), before any request can be made.
func Install() {
	once.Do(func() {
		allowed := allowedHosts()

		transport, ok := http.DefaultTransport.(*http.Transport)
		if ok {
			transport = transport.Clone()
		} else {
			transport = &http.Transport{}
		}

		// Never route through an HTTP(S) proxy: a proxy host is itself loopback/allowed in
		// many setups, so DialContext would only see the proxy address while the proxy
		// tunneled anywhere. Forcing Proxy=nil makes DialContext see the real destination.
		transport.Proxy = nil

		dialer := &net.Dialer{}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				host = addr
			}
			if _, allow := allowed[strings.ToLower(host)]; !allow {
				return nil, fmt.Errorf("netguard: blocked outbound connection to %q; this build may only reach %s", addr, strings.Join(sortedKeys(allowed), ", "))
			}
			return dialer.DialContext(ctx, network, addr)
		}

		// Force HTTP/1.1. Go's bundled HTTP/2 transport dials TLS on its own and would
		// bypass DialContext above, letting HTTPS requests escape the allowlist. A non-nil
		// (empty) TLSNextProto disables the automatic HTTP/2 upgrade so every request — and
		// every clone of this transport (the agent tools clone it) — goes through DialContext.
		transport.ForceAttemptHTTP2 = false
		transport.TLSNextProto = map[string]func(authority string, c *tls.Conn) http.RoundTripper{}

		http.DefaultTransport = transport
		http.DefaultClient = &http.Client{Transport: transport}
	})
}
