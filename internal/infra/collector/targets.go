package collector

import (
	"fmt"
	"net/url"
	"strings"
)

// ParseHTTPTargets converte uma string no formato "nome=url,nome=url" em uma lista de HTTPTarget.
func ParseHTTPTargets(raw string) ([]HTTPTarget, error) {
	var targets []HTTPTarget
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		target, err := parseHTTPTarget(entry)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func parseHTTPTarget(entry string) (HTTPTarget, error) {
	name, rawURL, found := strings.Cut(entry, "=")
	name, rawURL = strings.TrimSpace(name), strings.TrimSpace(rawURL)
	if !found || name == "" || rawURL == "" {
		return HTTPTarget{}, fmt.Errorf("invalid target %q: expected format name=url", entry)
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return HTTPTarget{}, fmt.Errorf("invalid URL for target %q: %q", name, rawURL)
	}
	return HTTPTarget{Name: name, URL: rawURL}, nil
}
