package helper

import "net/url"

// ExtractHostname parses a URL and returns just the hostname.
// Returns empty string if parsing fails or the URL has no hostname
// (for example a bare hostname without a scheme).
func ExtractHostname(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}
	return ""
}

// NormalizeToHostname reduces raw to its hostname when raw is a URL, and
// returns raw unchanged when it already is a bare hostname.
func NormalizeToHostname(raw string) string {
	if hostname := ExtractHostname(raw); hostname != "" {
		return hostname
	}
	return raw
}
