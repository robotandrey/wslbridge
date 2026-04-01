package db

import (
	"reflect"
	"testing"
)

// TestNormalizeServiceNames verifies trimming and case-insensitive deduplication.
func TestNormalizeServiceNames(t *testing.T) {
	got := normalizeServiceNames([]string{" example-db ", "reporting-db", "EXAMPLE-DB", "", "reporting-db"})
	want := []string{"example-db", "reporting-db"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeServiceNames got %v, want %v", got, want)
	}
}

// TestUpsertServiceName verifies idempotent insertion.
func TestUpsertServiceName(t *testing.T) {
	got := upsertServiceName([]string{"example-db"}, "EXAMPLE-DB")
	want := []string{"example-db"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("upsertServiceName got %v, want %v", got, want)
	}

	got = upsertServiceName(got, "analytics-db")
	want = []string{"example-db", "analytics-db"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("upsertServiceName got %v, want %v", got, want)
	}
}

// TestRemoveServiceName verifies case-insensitive removal.
func TestRemoveServiceName(t *testing.T) {
	got, removed := removeServiceName([]string{"example-db", "analytics-db"}, "EXAMPLE-DB")
	if !removed {
		t.Fatalf("removeServiceName expected removed=true")
	}
	want := []string{"analytics-db"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("removeServiceName got %v, want %v", got, want)
	}
}

// TestNormalizeServiceValues verifies filtering by service set and value trimming.
func TestNormalizeServiceValues(t *testing.T) {
	got := normalizeServiceValues(
		[]string{"example-db", "analytics-db"},
		map[string]string{
			"EXAMPLE-DB":   " 10.0.0.1:6432 ",
			"analytics-db": "db-analytics-db",
			"unknown":      "10.0.0.9:6432",
			"empty":        "   ",
		},
	)
	want := map[string]string{
		"example-db":   "10.0.0.1:6432",
		"analytics-db": "db-analytics-db",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeServiceValues got %v, want %v", got, want)
	}
}

// TestServiceKey verifies case normalization.
func TestServiceKey(t *testing.T) {
	if got, want := serviceKey(" Example-DB "), "example-db"; got != want {
		t.Fatalf("serviceKey got %q, want %q", got, want)
	}
}

// TestFindProxyRoute verifies case-insensitive service lookup.
func TestFindProxyRoute(t *testing.T) {
	routes := proxyRoutesFile{
		Services: map[string]proxyRoute{
			"example-db": {
				Service:    "example-db",
				TargetAddr: "10.0.0.1:6432",
			},
		},
	}

	route, err := findProxyRoute(routes, "EXAMPLE-DB", "")
	if err != nil {
		t.Fatalf("findProxyRoute() error: %v", err)
	}
	if got, want := route.TargetAddr, "10.0.0.1:6432"; got != want {
		t.Fatalf("findProxyRoute() target got %q, want %q", got, want)
	}
}
