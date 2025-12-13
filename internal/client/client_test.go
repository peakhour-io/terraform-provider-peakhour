package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test 1: Post should decode response body on HTTP 201 Created
func TestPost_DecodesResponseOn201Created(t *testing.T) {
	// Arrange: server returns 201 with JSON body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"name": "example.com",
			"uuid": "abc-123",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", server.URL)

	// Act
	var result struct {
		Name string `json:"name"`
		UUID string `json:"uuid"`
	}
	err := client.Post("/api/v1/domains", map[string]string{"name": "example.com"}, &result)

	// Assert
	if err != nil {
		t.Fatalf("Post() returned error: %v", err)
	}
	if result.Name != "example.com" {
		t.Errorf("Post() result.Name = %q, want %q", result.Name, "example.com")
	}
	if result.UUID != "abc-123" {
		t.Errorf("Post() result.UUID = %q, want %q", result.UUID, "abc-123")
	}
}

// Test 2: Post should decode response body on HTTP 200 OK
func TestPost_DecodesResponseOn200OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", server.URL)

	var result struct {
		Status string `json:"status"`
	}
	err := client.Post("/test", nil, &result)

	if err != nil {
		t.Fatalf("Post() returned error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("Post() result.Status = %q, want %q", result.Status, "ok")
	}
}

// Test 3: APIError should be returned with status code on 4xx/5xx
func TestClient_ReturnsAPIErrorWithStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"400 Bad Request", http.StatusBadRequest},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(`{"error": "something went wrong"}`))
			}))
			defer server.Close()

			client := NewClient("test-api-key", server.URL)
			err := client.Get("/test", nil)

			if err == nil {
				t.Fatal("Get() should return error for status", tt.statusCode)
			}

			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("Get() error type = %T, want *APIError", err)
			}
			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("APIError.StatusCode = %d, want %d", apiErr.StatusCode, tt.statusCode)
			}
		})
	}
}

// Test 4: IsNotFound helper should work correctly
func TestAPIError_IsNotFound(t *testing.T) {
	tests := []struct {
		statusCode int
		want       bool
	}{
		{http.StatusNotFound, true},
		{http.StatusBadRequest, false},
		{http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.statusCode), func(t *testing.T) {
			err := &APIError{StatusCode: tt.statusCode, Body: "test"}
			if got := err.IsNotFound(); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test 5: IsNotFound helper function should detect 404 from error
func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"404 APIError", &APIError{StatusCode: 404}, true},
		{"400 APIError", &APIError{StatusCode: 400}, false},
		{"generic error", context.Canceled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundError(tt.err); got != tt.want {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test 6: Client methods should accept context for cancellation
func TestClient_MethodsAcceptContext(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that request has context
		if r.Context() == nil {
			t.Error("Request should have context")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", server.URL)
	ctx := context.Background()

	// Test GetWithContext
	var result struct {
		Status string `json:"status"`
	}
	err := client.GetWithContext(ctx, "/test", &result)
	if err != nil {
		t.Errorf("GetWithContext() error = %v", err)
	}

	// Test PostWithContext
	err = client.PostWithContext(ctx, "/test", nil, &result)
	if err != nil {
		t.Errorf("PostWithContext() error = %v", err)
	}
}

// Test 7: Cancelled context should abort request
func TestClient_CancelledContextAbortsRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This should not complete if context is cancelled
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-api-key", server.URL)

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.GetWithContext(ctx, "/test", nil)
	if err == nil {
		t.Error("GetWithContext() should return error for cancelled context")
	}
}

// Test 8: Authorization header should be set
func TestClient_SetsAuthorizationHeader(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("my-secret-key", server.URL)
	client.Get("/test", nil)

	if capturedAuth != "Bearer my-secret-key" {
		t.Errorf("Authorization header = %q, want %q", capturedAuth, "Bearer my-secret-key")
	}
}

// Test 9: Rules endpoints should use phase in path
func TestClient_RulesPathIncludesPhase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/domains/example.com/services/rp/rules/phases/request/rule/abc-123"
		if r.URL.Path != expectedPath {
			t.Errorf("Path = %q, want %q", r.URL.Path, expectedPath)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uuid":  "abc-123",
			"phase": "request",
		})
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL)
	rule, err := client.GetRule("example.com", "request", "abc-123")
	if err != nil {
		t.Fatalf("GetRule failed: %v", err)
	}
	if rule.UUID != "abc-123" {
		t.Errorf("Rule UUID = %q, want %q", rule.UUID, "abc-123")
	}
}

// Test 10: Transform config endpoints
func TestClient_TransformConfigEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/domains/example.com/services/rp/config":
			if r.Method != "GET" {
				t.Errorf("Expected GET for config")
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"config": map[string]interface{}{
					"websocket": true,
				},
			})
		case "/api/v1/domains/example.com/image-transforms":
			if r.Method == "GET" {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"presets": []map[string]interface{}{
						{"uuid": "preset-1", "name": "Optimise"},
					},
				})
			} else if r.Method == "POST" {
				// Create
				var body ImageTransformPresetCreate
				json.NewDecoder(r.Body).Decode(&body)
				if body.Name != "NewPreset" {
					t.Errorf("Expected Name NewPreset, got %s", body.Name)
				}
				json.NewEncoder(w).Encode(map[string]interface{}{
					"uuid":   "preset-new",
					"name":   body.Name,
					"config": body.Config,
				})
			}
		case "/api/v1/domains/example.com/image-transforms/preset-1":
			if r.Method == "DELETE" {
				w.WriteHeader(http.StatusNoContent)
			} else if r.Method == "POST" {
				// Update
				var body ImageTransformPresetUpdate
				json.NewDecoder(r.Body).Decode(&body)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"uuid":   "preset-1",
					"name":   "Optimise",
					"config": body.Config,
				})
			}
		case "/api/v1/domains/example.com/image-transforms/config/commit":
			if r.Method != "POST" {
				t.Errorf("Expected POST for image transforms commit")
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL)

	// Test GetRPConfig
	rpConfig, err := client.GetRPConfig("example.com")
	if err != nil {
		t.Fatalf("GetRPConfig failed: %v", err)
	}
	if rpConfig.Config["websocket"] != true {
		t.Errorf("Expected Config[websocket] to be true, got %v", rpConfig.Config["websocket"])
	}

	// Test ListImageTransformPresets
	presets, err := client.ListImageTransformPresets("example.com")
	if err != nil {
		t.Fatalf("ListImageTransformPresets failed: %v", err)
	}
	if len(presets) != 1 || presets[0].Name != "Optimise" {
		t.Error("Expected 1 preset with name Optimise")
	}

	// Test CreateImageTransformPreset
	newPreset, err := client.CreateImageTransformPreset("example.com", ImageTransformPresetCreate{
		Name:   "NewPreset",
		Config: map[string]interface{}{"width": 100},
	})
	if err != nil {
		t.Fatalf("CreateImageTransformPreset failed: %v", err)
	}
	if newPreset.UUID != "preset-new" {
		t.Error("Expected new UUID preset-new")
	}

	// Test UpdateImageTransformPreset
	updatedPreset, err := client.UpdateImageTransformPreset("example.com", "preset-1", ImageTransformPresetUpdate{
		Config: map[string]interface{}{"width": 200},
	})
	if err != nil {
		t.Fatalf("UpdateImageTransformPreset failed: %v", err)
	}
	if updatedPreset.Name != "Optimise" {
		t.Error("Expected name Optimise")
	}

	// Test DeleteImageTransformPreset
	err = client.DeleteImageTransformPreset("example.com", "preset-1")
	if err != nil {
		t.Fatalf("DeleteImageTransformPreset failed: %v", err)
	}

	// Test CommitImageTransforms
	err = client.CommitImageTransforms("example.com")
	if err != nil {
		t.Fatalf("CommitImageTransforms failed: %v", err)
	}
}

// Test 11: Bulk redirect endpoints should use spec paths
func TestClient_BulkRedirectEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/domains/example.com/services/rp/rules/bulk_redirects":
			if r.Method == "GET" {
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{"uuid": "list-1", "name": "legacy", "description": nil, "entries_count": 0},
				})
				return
			}
			if r.Method == "POST" {
				var body BulkRedirectListCreate
				json.NewDecoder(r.Body).Decode(&body)
				if body.Name == nil || *body.Name == "" {
					t.Errorf("Expected non-empty list name in request body")
				}
				json.NewEncoder(w).Encode(map[string]interface{}{"uuid": "list-1"})
				return
			}
		case "/api/v1/domains/example.com/services/rp/rules/bulk_redirects/list-1":
			switch r.Method {
			case "GET":
				json.NewEncoder(w).Encode(map[string]interface{}{
					"uuid":          "list-1",
					"name":          "legacy",
					"description":   nil,
					"entries_count": 1,
				})
			case "PATCH":
				w.WriteHeader(http.StatusOK)
			case "DELETE":
				w.WriteHeader(http.StatusNoContent)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		case "/api/v1/domains/example.com/services/rp/rules/bulk_redirects/list-1/entries":
			if r.Method == "GET" {
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{"id": "entry-1", "source_path": "/old", "target_url": "https://example.com/new"},
				})
				return
			}
			if r.Method == "POST" {
				var body BulkRedirectEntryCreate
				json.NewDecoder(r.Body).Decode(&body)
				if body.SourcePath == nil || *body.SourcePath == "" {
					t.Errorf("Expected source_path in request body")
				}
				if body.TargetURL == nil || *body.TargetURL == "" {
					t.Errorf("Expected target_url in request body")
				}
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":          "entry-1",
					"source_path": body.SourcePath,
					"target_url":  body.TargetURL,
				})
				return
			}
		case "/api/v1/domains/example.com/services/rp/rules/bulk_redirects/list-1/entries/entry-1":
			switch r.Method {
			case "PATCH":
				w.WriteHeader(http.StatusOK)
			case "DELETE":
				w.WriteHeader(http.StatusNoContent)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		default:
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := NewClient("test-key", server.URL)

	if _, err := c.ListBulkRedirectLists("example.com"); err != nil {
		t.Fatalf("ListBulkRedirectLists failed: %v", err)
	}

	listName := "legacy"
	if _, err := c.CreateBulkRedirectList("example.com", BulkRedirectListCreate{Name: &listName}); err != nil {
		t.Fatalf("CreateBulkRedirectList failed: %v", err)
	}

	if _, err := c.GetBulkRedirectList("example.com", "list-1"); err != nil {
		t.Fatalf("GetBulkRedirectList failed: %v", err)
	}

	if err := c.UpdateBulkRedirectList("example.com", "list-1", BulkRedirectListUpdate{}); err != nil {
		t.Fatalf("UpdateBulkRedirectList failed: %v", err)
	}

	if _, err := c.ListBulkRedirectEntries("example.com", "list-1"); err != nil {
		t.Fatalf("ListBulkRedirectEntries failed: %v", err)
	}

	sourcePath := "/old"
	targetURL := "https://example.com/new"
	if _, err := c.CreateBulkRedirectEntry("example.com", "list-1", BulkRedirectEntryCreate{SourcePath: &sourcePath, TargetURL: &targetURL}); err != nil {
		t.Fatalf("CreateBulkRedirectEntry failed: %v", err)
	}

	if err := c.UpdateBulkRedirectEntry("example.com", "list-1", "entry-1", BulkRedirectEntryUpdate{}); err != nil {
		t.Fatalf("UpdateBulkRedirectEntry failed: %v", err)
	}

	if err := c.DeleteBulkRedirectEntry("example.com", "list-1", "entry-1"); err != nil {
		t.Fatalf("DeleteBulkRedirectEntry failed: %v", err)
	}

	if err := c.DeleteBulkRedirectList("example.com", "list-1"); err != nil {
		t.Fatalf("DeleteBulkRedirectList failed: %v", err)
	}
}

// Test 12: Config endpoints should use spec paths/methods
func TestClient_ConfigEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/domains/example.com/services/rp/settings":
			switch r.Method {
			case "GET":
				json.NewEncoder(w).Encode(map[string]interface{}{
					"notification_emails": []string{"ops@example.com"},
					"quickstart":          true,
					"ipv4_address":        "192.0.2.10",
					"ipv6_address":        "2001:db8::10",
					"cname":               "example.peakhour.io",
				})
			case "PATCH":
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return

		case "/api/v1/domains/example.com/services/rp/ssl":
			switch r.Method {
			case "GET":
				json.NewEncoder(w).Encode(map[string]interface{}{"ciphers": "intermediate"})
			case "PUT":
				var body SSLConfig
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body.Ciphers == "" {
					t.Errorf("expected ciphers in request body")
				}
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return

		case "/api/v1/domains/example.com/services/acme/settings":
			switch r.Method {
			case "GET":
				json.NewEncoder(w).Encode(map[string]any{
					"domain_names": []string{"example.com", "www.example.com"},
				})
			case "PUT":
				var body AcmeSettings
				_ = json.NewDecoder(r.Body).Decode(&body)
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return

		case "/api/v1/domains/example.com/services/rp/origin":
			switch r.Method {
			case "GET":
				json.NewEncoder(w).Encode(map[string]any{
					"ssl_mode":        "https",
					"origin_downtime": nil,
					"origin_request_headers": map[string]any{
						"blocklists":   true,
						"client_proxy": nil,
						"geoip":        false,
					},
				})
			case "POST":
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return

		case "/api/v1/domains/example.com/services/rp/cdn":
			switch r.Method {
			case "GET":
				json.NewEncoder(w).Encode(map[string]any{
					"cache_enabled":              true,
					"cdn_query_mode":             "full",
					"cdn_remove_query_args":      []string{"utm_source"},
					"cache_tag_header":           "X-Cache-Tag",
					"cache_tag_header_separator": ",",
				})
			case "PATCH":
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return

		case "/api/v1/domains/example.com/services/rp/bots":
			switch r.Method {
			case "GET":
				json.NewEncoder(w).Encode(map[string]any{
					"bots_inject_js":     true,
					"bots_verify_list":   []string{"google"},
					"bots_verify_rdns":   true,
					"bots_verify_invert": nil,
				})
			case "PATCH":
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return

		case "/api/v1/domains/example.com/services/rp/firewall":
			switch r.Method {
			case "GET":
				json.NewEncoder(w).Encode(map[string]any{
					"challenge_cookie_key": []string{"ip"},
				})
			case "POST":
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return

		case "/api/v1/domains/example.com/services/rp/firewall/error_page":
			switch r.Method {
			case "GET":
				json.NewEncoder(w).Encode(map[string]any{
					"error_page": true,
				})
			case "PUT":
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return

		case "/api/v1/domains/example.com/services/rp/lua":
			switch r.Method {
			case "GET":
				json.NewEncoder(w).Encode(map[string]any{
					"lua_enabled":                true,
					"lua_request_filter":         "return true",
					"lua_response_filter":        nil,
					"lua_origin_request_filter":  nil,
					"lua_origin_response_filter": nil,
					"lua_origin_selector":        nil,
					"lua_origin_pool_selector":   nil,
				})
			case "PUT":
				var body LuaOptions
				_ = json.NewDecoder(r.Body).Decode(&body)
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return

		case "/api/v1/domains/example.com/services/rp/rate_limit":
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			var body RateLimitSettings
			_ = json.NewDecoder(r.Body).Decode(&body)
			if len(body.Mode) == 0 {
				t.Errorf("expected mode in request body")
			}
			w.WriteHeader(http.StatusOK)
			return

		default:
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := NewClient("test-key", server.URL)

	if _, err := c.GetRPSettings("example.com"); err != nil {
		t.Fatalf("GetRPSettings failed: %v", err)
	}
	if err := c.UpdateRPSettings("example.com", map[string]any{"quickstart": nil}); err != nil {
		t.Fatalf("UpdateRPSettings failed: %v", err)
	}

	if _, err := c.GetRPSSLConfig("example.com"); err != nil {
		t.Fatalf("GetRPSSLConfig failed: %v", err)
	}
	if err := c.UpdateRPSSLConfig("example.com", SSLConfig{Ciphers: "modern"}); err != nil {
		t.Fatalf("UpdateRPSSLConfig failed: %v", err)
	}

	if _, err := c.GetAcmeSettings("example.com"); err != nil {
		t.Fatalf("GetAcmeSettings failed: %v", err)
	}
	if err := c.UpdateAcmeSettings("example.com", AcmeSettings{DomainNames: []string{"example.com"}}); err != nil {
		t.Fatalf("UpdateAcmeSettings failed: %v", err)
	}

	if _, err := c.GetRPOriginConfig("example.com"); err != nil {
		t.Fatalf("GetRPOriginConfig failed: %v", err)
	}
	if err := c.UpdateRPOriginConfig("example.com", map[string]any{"ssl_mode": "https"}); err != nil {
		t.Fatalf("UpdateRPOriginConfig failed: %v", err)
	}

	if _, err := c.GetRPCDNCacheConfig("example.com"); err != nil {
		t.Fatalf("GetRPCDNCacheConfig failed: %v", err)
	}
	if err := c.UpdateRPCDNCacheConfig("example.com", map[string]any{"cache_enabled": true}); err != nil {
		t.Fatalf("UpdateRPCDNCacheConfig failed: %v", err)
	}

	if _, err := c.GetRPBotsConfig("example.com"); err != nil {
		t.Fatalf("GetRPBotsConfig failed: %v", err)
	}
	if err := c.UpdateRPBotsConfig("example.com", map[string]any{"bots_verify_rdns": false}); err != nil {
		t.Fatalf("UpdateRPBotsConfig failed: %v", err)
	}

	if _, err := c.GetRPFirewallSettings("example.com"); err != nil {
		t.Fatalf("GetRPFirewallSettings failed: %v", err)
	}
	if err := c.UpdateRPFirewallSettings("example.com", map[string]any{"challenge_cookie_key": []string{"ip"}}); err != nil {
		t.Fatalf("UpdateRPFirewallSettings failed: %v", err)
	}

	if _, err := c.GetRPFirewallErrorPage("example.com"); err != nil {
		t.Fatalf("GetRPFirewallErrorPage failed: %v", err)
	}
	if err := c.UpdateRPFirewallErrorPage("example.com", map[string]any{"error_page": nil}); err != nil {
		t.Fatalf("UpdateRPFirewallErrorPage failed: %v", err)
	}

	if _, err := c.GetRPLuaOptions("example.com"); err != nil {
		t.Fatalf("GetRPLuaOptions failed: %v", err)
	}
	if err := c.UpdateRPLuaOptions("example.com", LuaOptions{}); err != nil {
		t.Fatalf("UpdateRPLuaOptions failed: %v", err)
	}

	if err := c.UpdateRateLimitSettings("example.com", RateLimitSettings{Mode: []string{"zone"}}); err != nil {
		t.Fatalf("UpdateRateLimitSettings failed: %v", err)
	}
}
