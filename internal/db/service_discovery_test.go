package db

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestChooseEndpoint_PreferRole verifies role-first selection.
func TestChooseEndpoint_PreferRole(t *testing.T) {
	endpoints := []Endpoint{
		{InstanceName: "a", Address: "10.0.0.1:6432", Role: "sync", IsDefaultRoute: true},
		{InstanceName: "b", Address: "10.0.0.2:6432", Role: "master", IsDefaultRoute: true},
	}
	got, err := ChooseEndpoint(endpoints, "master")
	if err != nil {
		t.Fatalf("ChooseEndpoint error: %v", err)
	}
	if got.InstanceName != "b" {
		t.Fatalf("ChooseEndpoint picked %q, want %q", got.InstanceName, "b")
	}
}

// TestChooseEndpoint_DefaultRouteFallback verifies default-route fallback.
func TestChooseEndpoint_DefaultRouteFallback(t *testing.T) {
	endpoints := []Endpoint{
		{InstanceName: "a", Address: "10.0.0.1:6432", Role: "sync", IsDefaultRoute: false},
		{InstanceName: "b", Address: "10.0.0.2:6432", Role: "sync", IsDefaultRoute: true},
	}
	got, err := ChooseEndpoint(endpoints, "master")
	if err != nil {
		t.Fatalf("ChooseEndpoint error: %v", err)
	}
	if got.InstanceName != "b" {
		t.Fatalf("ChooseEndpoint picked %q, want %q", got.InstanceName, "b")
	}
}

// TestChooseEndpoint_EmptyAddress verifies rejection of empty addresses.
func TestChooseEndpoint_EmptyAddress(t *testing.T) {
	endpoints := []Endpoint{
		{InstanceName: "a", Address: "", Role: "master", IsDefaultRoute: true},
	}
	if _, err := ChooseEndpoint(endpoints, "master"); err == nil {
		t.Fatalf("ChooseEndpoint expected error for empty address")
	}
}

// TestFetchEndpoints validates Service discovery response parsing.
func TestFetchEndpoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"InstanceName":"db-a","Address":"10.0.0.1:6432","Role":"master","IsDefaultRoute":true}]`))
	}))
	defer srv.Close()

	endpoints, err := FetchEndpoints(srv.URL)
	if err != nil {
		t.Fatalf("FetchEndpoints error: %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("FetchEndpoints len=%d, want 1", len(endpoints))
	}
	if endpoints[0].Address != "10.0.0.1:6432" {
		t.Fatalf("FetchEndpoints address=%q", endpoints[0].Address)
	}
}

// TestFetchEndpoints_BadStatus validates non-2xx handling.
func TestFetchEndpoints_BadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	if _, err := FetchEndpoints(srv.URL); err == nil {
		t.Fatalf("FetchEndpoints expected error for non-2xx status")
	}
}

// TestNormalizeServiceDiscoveryInput_HostOnly verifies host-only input normalization.
func TestNormalizeServiceDiscoveryInput_HostOnly(t *testing.T) {
	scheme, host, err := NormalizeServiceDiscoveryInput("service-discovery.example.internal")
	if err != nil {
		t.Fatalf("NormalizeServiceDiscoveryInput error: %v", err)
	}
	if scheme != "http" || host != "service-discovery.example.internal" {
		t.Fatalf("NormalizeServiceDiscoveryInput got %q %q", scheme, host)
	}
}

// TestNormalizeServiceDiscoveryInput_FullURL verifies URL with path normalization.
func TestNormalizeServiceDiscoveryInput_FullURL(t *testing.T) {
	scheme, host, err := NormalizeServiceDiscoveryInput("https://service-discovery.example.internal/endpoints?service=example-db.pg:bouncer")
	if err != nil {
		t.Fatalf("NormalizeServiceDiscoveryInput error: %v", err)
	}
	if scheme != "https" || host != "service-discovery.example.internal" {
		t.Fatalf("NormalizeServiceDiscoveryInput got %q %q", scheme, host)
	}
}

// TestNormalizeEndpointMask_AbsoluteURL verifies absolute URL mask normalization.
func TestNormalizeEndpointMask_AbsoluteURL(t *testing.T) {
	mask, err := NormalizeEndpointMask("http://service-discovery.example.internal/endpoints?service=%s.pg:bouncer")
	if err != nil {
		t.Fatalf("NormalizeEndpointMask error: %v", err)
	}
	if mask != "/endpoints?service=%s.pg:bouncer" {
		t.Fatalf("NormalizeEndpointMask got %q", mask)
	}
}

// TestNormalizeEndpointMask_MissingPlaceholder verifies validation.
func TestNormalizeEndpointMask_MissingPlaceholder(t *testing.T) {
	if _, err := NormalizeEndpointMask("/endpoints?service=example-db.pg:bouncer"); err == nil {
		t.Fatalf("NormalizeEndpointMask expected error for missing placeholder")
	}
}

// TestBuildEndpointURL verifies final URL assembly from host + mask + service.
func TestBuildEndpointURL(t *testing.T) {
	url, err := BuildEndpointURL(
		"http",
		"service-discovery.example.internal",
		"/endpoints?service=<database>.pg:bouncer",
		"example-db",
	)
	if err != nil {
		t.Fatalf("BuildEndpointURL error: %v", err)
	}
	want := "http://service-discovery.example.internal/endpoints?service=example-db.pg:bouncer"
	if url != want {
		t.Fatalf("BuildEndpointURL got %q, want %q", url, want)
	}
}

// TestRenderEndpointMask_EscapesService verifies URL-escaping for service value.
func TestRenderEndpointMask_EscapesService(t *testing.T) {
	got, err := RenderEndpointMask("/endpoints?service=%s.pg:bouncer", "service with space")
	if err != nil {
		t.Fatalf("RenderEndpointMask error: %v", err)
	}
	if !strings.Contains(got, "service+with+space.pg:bouncer") {
		t.Fatalf("RenderEndpointMask got %q", got)
	}
}

// TestExtractEndpointMaskFromServiceDiscoveryInput_FullURL verifies mask extraction from full URL with service.
func TestExtractEndpointMaskFromServiceDiscoveryInput_FullURL(t *testing.T) {
	mask, ok, err := ExtractEndpointMaskFromServiceDiscoveryInput("http://service-discovery.example.internal/endpoints?service=example-db.pg:bouncer")
	if err != nil {
		t.Fatalf("ExtractEndpointMaskFromServiceDiscoveryInput error: %v", err)
	}
	if !ok {
		t.Fatalf("ExtractEndpointMaskFromServiceDiscoveryInput expected ok=true")
	}
	if mask != "/endpoints?service=<db>.pg:bouncer" {
		t.Fatalf("ExtractEndpointMaskFromServiceDiscoveryInput mask=%q", mask)
	}
}

// TestExtractEndpointMaskFromServiceDiscoveryInput_HostOnly verifies no mask extraction for host-only input.
func TestExtractEndpointMaskFromServiceDiscoveryInput_HostOnly(t *testing.T) {
	mask, ok, err := ExtractEndpointMaskFromServiceDiscoveryInput("service-discovery.example.internal")
	if err != nil {
		t.Fatalf("ExtractEndpointMaskFromServiceDiscoveryInput error: %v", err)
	}
	if ok || mask != "" {
		t.Fatalf("ExtractEndpointMaskFromServiceDiscoveryInput got ok=%v mask=%q, want ok=false mask=\"\"", ok, mask)
	}
}

// TestExtractEndpointMaskFromServiceDiscoveryInput_Placeholder verifies preserving explicit placeholder.
func TestExtractEndpointMaskFromServiceDiscoveryInput_Placeholder(t *testing.T) {
	mask, ok, err := ExtractEndpointMaskFromServiceDiscoveryInput("http://service-discovery.example.internal/endpoints?service=%s.pg:bouncer")
	if err != nil {
		t.Fatalf("ExtractEndpointMaskFromServiceDiscoveryInput error: %v", err)
	}
	if !ok {
		t.Fatalf("ExtractEndpointMaskFromServiceDiscoveryInput expected ok=true")
	}
	if mask != "/endpoints?service=%s.pg:bouncer" {
		t.Fatalf("ExtractEndpointMaskFromServiceDiscoveryInput mask=%q", mask)
	}
}
