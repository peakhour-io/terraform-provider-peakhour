package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

func TestReverseProxyConfigResource_deleteConfig_ResetsConfig(t *testing.T) {
	// Setup mock server
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" && r.URL.Path == "/api/v1/domains/example.com/services/rp" {
			called = true
			// Check body contains explicit resets
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("Failed to decode body: %v", err)
			}

			// Check for presence of key 'websocket' with value false
			if v, ok := body["websocket"]; !ok || v != false {
				t.Errorf("Expected websocket: false, got %v", v)
			}
			// Check for 'aliases' with empty list
			if v, ok := body["aliases"]; !ok {
				t.Errorf("Expected aliases key to be present")
			} else {
				// json decodes empty array to []interface{}
				list, ok := v.([]interface{})
				if !ok || len(list) != 0 {
					t.Errorf("Expected aliases to be [], got %v", v)
				}
			}
			// Just checking a few representative fields is enough for TDD cycle
			if v, ok := body["redirect_mode"]; !ok || v != "none" {
				t.Errorf("Expected redirect_mode: \"none\", got %v", v)
			}
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Setup client
	c := client.NewClient("fake-key", server.URL)
	// Replace default HTTP client with test server's client or just use server.URL which we did.

	// Setup resource
	r := &ReverseProxyConfigResource{}
	r.client = c // Available since we are in provider package

	// Call deleteConfig (to be implemented)
	err := r.deleteConfig(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("deleteConfig failed: %v", err)
	}

	if !called {
		t.Error("Expected PATCH request to reset config, but none received")
	}
}

func TestReverseProxyConfigResource_mapConfigToModel_Nulls(t *testing.T) {
	// API returns nil for optional fields
	apiConfig := &client.ReverseProxyConfig{
		Websocket:          nil,
		Gzip:               nil,
		Brotli:             nil,
		TrackSessions:      nil,
		Debug:              nil,
		Segment:            nil,
		RedirectMode:       nil,
		RedirectLocation:   nil,
		RedirectStatusCode: nil,
		Aliases:            nil,
	}

	// Model has previous values
	model := ReverseProxyConfigResourceModel{
		Domain:             types.StringValue("example.com"),
		Websocket:          types.BoolValue(true),
		Gzip:               types.BoolValue(true),
		Brotli:             types.BoolValue(true),
		TrackSessions:      types.BoolValue(true),
		Debug:              types.BoolValue(true),
		Segment:            types.BoolValue(true),
		RedirectMode:       types.StringValue("http"),
		RedirectLocation:   types.StringValue("/foo"),
		RedirectStatusCode: types.Int64Value(302),
		// Aliases omitted for simplicity as list setup is verbose; focussing on scalars for null logic
	}

	r := &ReverseProxyConfigResource{}
	r.mapConfigToModel(context.Background(), apiConfig, &model)

	if !model.Websocket.IsNull() {
		t.Errorf("Expected Websocket to be Null, got %v", model.Websocket)
	}
	if !model.Gzip.IsNull() {
		t.Errorf("Expected Gzip to be Null, got %v", model.Gzip)
	}
	if !model.RedirectMode.IsNull() {
		t.Errorf("Expected RedirectMode to be Null, got %v", model.RedirectMode)
	}
	if !model.RedirectStatusCode.IsNull() {
		t.Errorf("Expected RedirectStatusCode to be Null, got %v", model.RedirectStatusCode)
	}
	if !model.Aliases.IsNull() {
		t.Errorf("Expected Aliases to be Null, got %v", model.Aliases)
	}
}
