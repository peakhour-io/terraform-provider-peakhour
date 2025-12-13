package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"testing"
)

func TestClient_ListDomains(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/api/v1/domains" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"domains": []map[string]any{
				{"name": "b.com"},
				{"name": "a.com"},
			},
			"granted_domains": []map[string]any{
				{"name": "c.com"},
				{"name": "a.com"}, // duplicate across owned/granted
			},
		})
	}))
	defer server.Close()

	c := NewClient("test-key", server.URL)

	method := reflect.ValueOf(c).MethodByName("ListDomains")
	if !method.IsValid() {
		t.Fatalf("Client missing ListDomains method")
	}

	out := method.Call(nil)
	if len(out) != 2 {
		t.Fatalf("ListDomains should return (domains, error), got %d values", len(out))
	}

	if !out[1].IsNil() {
		err, _ := out[1].Interface().(error)
		t.Fatalf("ListDomains returned error: %v", err)
	}

	got, ok := out[0].Interface().([]string)
	if !ok {
		t.Fatalf("ListDomains first return value should be []string, got %T", out[0].Interface())
	}

	want := []string{"a.com", "b.com", "c.com"}
	if !slices.Equal(got, want) {
		t.Fatalf("ListDomains returned %v, want %v", got, want)
	}
}

