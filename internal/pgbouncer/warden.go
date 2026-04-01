package pgbouncer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var maskPlaceholderRE = regexp.MustCompile(`<[^>]+>`)

// Endpoint describes a Warden endpoint entry.
type Endpoint struct {
	ReleaseName    string `json:"ReleaseName"`
	InstanceName   string `json:"InstanceName"`
	Address        string `json:"Address"`
	Version        string `json:"Version"`
	Role           string `json:"Role"`
	Weight         int    `json:"Weight"`
	IsDefaultRoute bool   `json:"IsDefaultRoute"`
}

// FetchEndpoints gets endpoint data from Warden.
func FetchEndpoints(wardenURL string) ([]Endpoint, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(wardenURL)
	if err != nil {
		return nil, fmt.Errorf("fetch warden endpoints: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("warden returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read warden response: %w", err)
	}

	var endpoints []Endpoint
	if err := json.Unmarshal(body, &endpoints); err != nil {
		return nil, fmt.Errorf("decode warden response: %w", err)
	}
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("warden returned no endpoints")
	}
	return endpoints, nil
}

// ChooseEndpoint selects a preferred endpoint by role/default route.
func ChooseEndpoint(endpoints []Endpoint, preferRole string) (Endpoint, error) {
	if len(endpoints) == 0 {
		return Endpoint{}, fmt.Errorf("no endpoints available")
	}

	role := strings.ToLower(strings.TrimSpace(preferRole))
	if role != "" && role != "any" {
		if ep, ok := pick(endpoints, func(e Endpoint) bool {
			return strings.EqualFold(e.Role, role) && e.IsDefaultRoute && e.Address != ""
		}); ok {
			return ep, nil
		}
		if ep, ok := pick(endpoints, func(e Endpoint) bool {
			return strings.EqualFold(e.Role, role) && e.Address != ""
		}); ok {
			return ep, nil
		}
	}

	if ep, ok := pick(endpoints, func(e Endpoint) bool { return e.IsDefaultRoute && e.Address != "" }); ok {
		return ep, nil
	}
	if ep, ok := pick(endpoints, func(e Endpoint) bool { return e.Address != "" }); ok {
		return ep, nil
	}

	return Endpoint{}, fmt.Errorf("no endpoint with non-empty address")
}

func pick(endpoints []Endpoint, match func(Endpoint) bool) (Endpoint, bool) {
	for _, e := range endpoints {
		if match(e) {
			return e, true
		}
	}
	return Endpoint{}, false
}

// NormalizeWardenInput converts an arbitrary Warden input to scheme and host.
func NormalizeWardenInput(raw string) (string, string, error) {
	val := strings.TrimSpace(raw)
	if val == "" {
		return "", "", fmt.Errorf("warden url must not be empty")
	}

	candidate := val
	if !strings.Contains(candidate, "://") {
		candidate = "http://" + candidate
	}
	u, err := url.Parse(candidate)
	if err != nil {
		return "", "", fmt.Errorf("invalid warden url: %w", err)
	}

	scheme := strings.ToLower(strings.TrimSpace(u.Scheme))
	if scheme == "" {
		scheme = "http"
	}
	if scheme != "http" && scheme != "https" {
		return "", "", fmt.Errorf("warden scheme must be http or https")
	}

	host := strings.ToLower(strings.TrimSpace(u.Host))
	if host == "" {
		return "", "", fmt.Errorf("warden host must not be empty")
	}

	return scheme, host, nil
}

// NormalizeEndpointMask validates endpoint mask and normalizes it to request URI form.
func NormalizeEndpointMask(mask string) (string, error) {
	val := strings.TrimSpace(mask)
	if val == "" {
		return "", fmt.Errorf("endpoint mask must not be empty")
	}

	lower := strings.ToLower(val)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		u, err := url.Parse(val)
		if err != nil {
			return "", fmt.Errorf("invalid endpoint mask URL: %w", err)
		}
		val = u.RequestURI()
	}

	if !strings.HasPrefix(val, "/") {
		val = "/" + val
	}
	if !hasMaskPlaceholder(val) {
		return "", fmt.Errorf("endpoint mask must contain a placeholder (`%%s` or `<...>`)")
	}
	return val, nil
}

// BuildEndpointURL builds final Warden endpoint URL from persisted host + mask + service.
func BuildEndpointURL(scheme, host, endpointMask, serviceName string) (string, error) {
	valScheme := strings.ToLower(strings.TrimSpace(scheme))
	if valScheme == "" {
		valScheme = "http"
	}
	if valScheme != "http" && valScheme != "https" {
		return "", fmt.Errorf("warden scheme must be http or https")
	}

	valHost := strings.TrimSpace(host)
	if valHost == "" {
		return "", fmt.Errorf("warden host must not be empty")
	}

	mask, err := NormalizeEndpointMask(endpointMask)
	if err != nil {
		return "", err
	}
	requestURI, err := RenderEndpointMask(mask, serviceName)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s%s", valScheme, valHost, requestURI), nil
}

// RenderEndpointMask injects service name into endpoint mask.
func RenderEndpointMask(endpointMask, serviceName string) (string, error) {
	mask := strings.TrimSpace(endpointMask)
	service := strings.TrimSpace(serviceName)
	if mask == "" {
		return "", fmt.Errorf("endpoint mask must not be empty")
	}
	if service == "" {
		return "", fmt.Errorf("service name must not be empty")
	}

	escaped := url.QueryEscape(service)
	if strings.Contains(mask, "%s") {
		return strings.ReplaceAll(mask, "%s", escaped), nil
	}

	loc := maskPlaceholderRE.FindStringIndex(mask)
	if loc == nil {
		return "", fmt.Errorf("endpoint mask must contain a placeholder (`%%s` or `<...>`)")
	}

	return mask[:loc[0]] + escaped + mask[loc[1]:], nil
}

func hasMaskPlaceholder(mask string) bool {
	return strings.Contains(mask, "%s") || maskPlaceholderRE.MatchString(mask)
}

// ExtractEndpointMaskFromWardenInput tries to derive endpoint mask from a full Warden URL.
func ExtractEndpointMaskFromWardenInput(raw string) (string, bool, error) {
	val := strings.TrimSpace(raw)
	if val == "" {
		return "", false, fmt.Errorf("warden url must not be empty")
	}

	candidate := val
	if !strings.Contains(candidate, "://") {
		candidate = "http://" + candidate
	}
	u, err := url.Parse(candidate)
	if err != nil {
		return "", false, fmt.Errorf("invalid warden url: %w", err)
	}

	hasPath := strings.TrimSpace(u.EscapedPath()) != "" && strings.TrimSpace(u.EscapedPath()) != "/"
	hasQuery := strings.TrimSpace(u.RawQuery) != ""
	if !hasPath && !hasQuery {
		return "", false, nil
	}

	requestURI := restoreMaskPlaceholders(u.RequestURI())
	if hasMaskPlaceholder(requestURI) {
		mask, err := NormalizeEndpointMask(requestURI)
		if err != nil {
			return "", false, err
		}
		return mask, true, nil
	}

	q := u.Query()
	serviceVal := strings.TrimSpace(q.Get("service"))
	if serviceVal == "" {
		return "", false, nil
	}

	q.Set("service", deriveServiceMaskValue(serviceVal))
	u.RawQuery = q.Encode()
	mask, err := NormalizeEndpointMask(restoreMaskPlaceholders(u.RequestURI()))
	if err != nil {
		return "", false, err
	}
	return mask, true, nil
}

func deriveServiceMaskValue(serviceVal string) string {
	val := strings.TrimSpace(serviceVal)
	if val == "" {
		return "<db>"
	}
	if hasMaskPlaceholder(val) {
		return val
	}
	if strings.Contains(val, "%s") {
		return val
	}
	if idx := strings.Index(val, "."); idx > 0 {
		return "<db>" + val[idx:]
	}
	return "<db>"
}

func restoreMaskPlaceholders(requestURI string) string {
	out := requestURI
	out = strings.ReplaceAll(out, url.QueryEscape("<db>"), "<db>")
	out = strings.ReplaceAll(out, url.QueryEscape("%s"), "%s")
	out = strings.ReplaceAll(out, "%3A", ":")
	out = strings.ReplaceAll(out, "%3a", ":")
	return out
}
