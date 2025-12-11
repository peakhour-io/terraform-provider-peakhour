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

func TestTransformSettingsResource_resetSettings(t *testing.T) {
	// Setup mock server
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" && r.URL.Path == "/api/v1/domains/example.com/services/rp/transforms" {
			called = true
			// Check body is empty JSON object "{}" with all fields null
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("Failed to decode body: %v", err)
			}
			if len(body) != 0 {
				t.Errorf("Expected empty body, got %v", body)
			}
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Setup client
	c := client.NewClient("fake-key", server.URL)

	// Setup resource
	r := &TransformSettingsResource{}
	r.client = c

	// Call resetSettings
	err := r.resetSettings(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("resetSettings failed: %v", err)
	}

	if !called {
		t.Error("Expected POST request to reset settings, but none received")
	}
}

func TestTransformSettingsResource_mapSettingsToModel_Nulls(t *testing.T) {
	// API returns nil for optional fields
	settings := &client.TransformSettings{
		TransformHTML:               nil,
		TransformBeacon:             nil,
		TransformLazySizes:          nil,
		TransformMixedContent:       nil,
		TransformImgDimsToQueryArgs: nil,
		TransformImageQuality:       nil,
		TransformImageFormat:        nil,
		TransformImageOptimise:      nil,
		TransformImageAPI:           nil,
		TransformHTTPHeaderValue:    nil,
		TransformESI:                nil,
		TransformRewriteDomains:     nil, // or empty slice
	}

	// Model has previous values
	model := TransformSettingsResourceModel{
		Domain:                   types.StringValue("example.com"),
		TransformHTML:            types.BoolValue(true),
		TransformHTTPHeaderValue: types.StringValue("foo"),
		TransformImageQuality:    types.Int64Value(90),
	}

	r := &TransformSettingsResource{}
	r.mapSettingsToModel(context.Background(), settings, &model)

	if !model.TransformHTML.IsNull() {
		t.Errorf("Expected TransformHTML to be Null, got %v", model.TransformHTML)
	}
	if !model.TransformHTTPHeaderValue.IsNull() {
		t.Errorf("Expected TransformHTTPHeaderValue to be Null, got %v", model.TransformHTTPHeaderValue)
	}
	if !model.TransformImageQuality.IsNull() {
		t.Errorf("Expected TransformImageQuality to be Null, got %v", model.TransformImageQuality)
	}
}
