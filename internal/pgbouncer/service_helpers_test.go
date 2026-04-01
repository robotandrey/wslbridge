package pgbouncer

import (
	"reflect"
	"testing"
)

// TestNormalizeServiceNames verifies trimming and case-insensitive deduplication.
func TestNormalizeServiceNames(t *testing.T) {
	got := normalizeServiceNames([]string{" chatapi-ng ", "saturn", "CHATAPI-NG", "", "saturn"})
	want := []string{"chatapi-ng", "saturn"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeServiceNames got %v, want %v", got, want)
	}
}

// TestUpsertServiceName verifies idempotent insertion.
func TestUpsertServiceName(t *testing.T) {
	got := upsertServiceName([]string{"chatapi-ng"}, "CHATAPI-NG")
	want := []string{"chatapi-ng"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("upsertServiceName got %v, want %v", got, want)
	}

	got = upsertServiceName(got, "bozon-saturn")
	want = []string{"chatapi-ng", "bozon-saturn"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("upsertServiceName got %v, want %v", got, want)
	}
}

// TestRemoveServiceName verifies case-insensitive removal.
func TestRemoveServiceName(t *testing.T) {
	got, removed := removeServiceName([]string{"chatapi-ng", "bozon-saturn"}, "CHATAPI-NG")
	if !removed {
		t.Fatalf("removeServiceName expected removed=true")
	}
	want := []string{"bozon-saturn"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("removeServiceName got %v, want %v", got, want)
	}
}

// TestNormalizeServiceValues verifies filtering by service set and value trimming.
func TestNormalizeServiceValues(t *testing.T) {
	got := normalizeServiceValues(
		[]string{"chatapi-ng", "bozon-saturn"},
		map[string]string{
			"CHATAPI-NG":   " 10.0.0.1:6432 ",
			"bozon-saturn": "db-bozon-saturn",
			"unknown":      "10.0.0.9:6432",
			"empty":        "   ",
		},
	)
	want := map[string]string{
		"chatapi-ng":   "10.0.0.1:6432",
		"bozon-saturn": "db-bozon-saturn",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeServiceValues got %v, want %v", got, want)
	}
}

// TestServiceKey verifies case normalization.
func TestServiceKey(t *testing.T) {
	if got, want := serviceKey(" ChatApi-NG "), "chatapi-ng"; got != want {
		t.Fatalf("serviceKey got %q, want %q", got, want)
	}
}

// TestFindProxyRoute verifies case-insensitive service lookup.
func TestFindProxyRoute(t *testing.T) {
	routes := proxyRoutesFile{
		Services: map[string]proxyRoute{
			"chatapi-ng": {
				Service:    "chatapi-ng",
				TargetAddr: "10.0.0.1:6432",
			},
		},
	}

	route, err := findProxyRoute(routes, "CHATAPI-NG", "")
	if err != nil {
		t.Fatalf("findProxyRoute() error: %v", err)
	}
	if got, want := route.TargetAddr, "10.0.0.1:6432"; got != want {
		t.Fatalf("findProxyRoute() target got %q, want %q", got, want)
	}
}
