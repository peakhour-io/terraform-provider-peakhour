package spec

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/provider"
)

type openAPISpec struct {
	Paths      map[string]map[string]any `json:"paths"`
	Components struct {
		Schemas map[string]any `json:"schemas"`
	} `json:"components"`
}

func loadSpec(t *testing.T) openAPISpec {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	path := filepath.Join(repoRoot, "docs", "spec", "peakhour-api-v1.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read spec %q: %v", path, err)
	}

	var spec openAPISpec
	if err := json.Unmarshal(b, &spec); err != nil {
		t.Fatalf("parse spec %q: %v", path, err)
	}
	return spec
}

func schemaProperties(t *testing.T, spec openAPISpec, schemaName string) map[string]any {
	t.Helper()

	raw, ok := spec.Components.Schemas[schemaName]
	if !ok {
		t.Fatalf("spec missing schema %q", schemaName)
	}

	obj, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("schema %q not an object", schemaName)
	}

	props, _ := obj["properties"].(map[string]any)
	return props
}

func TestSpecContract_RateLimitZone_NoConcurrentConnections(t *testing.T) {
	spec := loadSpec(t)

	props := schemaProperties(t, spec, "RateLimitZone")
	if _, ok := props["concurrent_connections"]; ok {
		t.Fatalf("spec RateLimitZone should not define concurrent_connections")
	}

	typ := reflect.TypeOf(client.RateLimitZone{})
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("json")
		if strings.HasPrefix(tag, "concurrent_connections") {
			t.Fatalf("client.RateLimitZone should not have json tag %q", tag)
		}
	}
}

func TestRateLimitZoneSchema_NoConcurrentConnections(t *testing.T) {
	r := provider.NewRateLimitZoneResource()

	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if _, ok := resp.Schema.Attributes["concurrent_connections"]; ok {
		t.Fatalf("rate_limit_zone schema should not expose concurrent_connections")
	}
}

func TestProviderResources_IncludeRateLimitGlobal(t *testing.T) {
	p := provider.New("test")()

	names := map[string]struct{}{}
	for _, factory := range p.Resources(context.Background()) {
		res := factory()
		var resp resource.MetadataResponse
		res.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "peakhour"}, &resp)
		names[resp.TypeName] = struct{}{}
	}

	if _, ok := names["peakhour_rate_limit_global"]; !ok {
		t.Fatalf("provider should register peakhour_rate_limit_global resource, got %v", sortedKeys(names))
	}
}

func TestRateLimitGlobalSchema_Attributes(t *testing.T) {
	res := getResourceByTypeName(t, "peakhour_rate_limit_global")

	var resp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes
	for _, name := range []string{
		"id",
		"domain",
		"concurrent_connections",
		"connections_max",
		"connections_interval_sec",
		"requests_max",
		"requests_interval_sec",
		"response_errors_max",
		"response_errors_interval_sec",
		"block_duration_sec",
	} {
		if _, ok := attrs[name]; !ok {
			t.Fatalf("rate_limit_global schema missing attribute %q", name)
		}
	}
	if _, ok := attrs["name"]; ok {
		t.Fatalf("rate_limit_global schema should not have attribute name")
	}
}

func TestClient_GetRPConfig_ReturnsRawConfig(t *testing.T) {
	spec := loadSpec(t)

	op, ok := spec.Paths["/api/v1/domains/{domain}/services/rp/config"]
	if !ok {
		t.Fatalf("spec missing /api/v1/domains/{domain}/services/rp/config path")
	}
	getOp, ok := op["get"].(map[string]any)
	if !ok {
		t.Fatalf("spec missing GET operation for rp/config")
	}
	responses, _ := getOp["responses"].(map[string]any)
	resp200, _ := responses["200"].(map[string]any)
	content, _ := resp200["content"].(map[string]any)
	appJSON, _ := content["application/json"].(map[string]any)
	schema, _ := appJSON["schema"].(map[string]any)
	ref, _ := schema["$ref"].(string)
	if !strings.HasSuffix(ref, "/RawConfig") {
		t.Fatalf("spec rp/config GET should return RawConfig, got $ref=%q", ref)
	}

	method, ok := reflect.TypeOf((*client.Client)(nil)).MethodByName("GetRPConfig")
	if !ok {
		t.Fatalf("client.Client missing GetRPConfig method")
	}
	if method.Type.NumOut() < 1 {
		t.Fatalf("client.Client.GetRPConfig has no return values")
	}

	out := method.Type.Out(0)
	if out.Kind() == reflect.Ptr {
		out = out.Elem()
	}
	field, ok := out.FieldByName("Config")
	if !ok {
		t.Fatalf("GetRPConfig should return a type with Config field (RawConfig), got %v", out.Name())
	}
	tag := field.Tag.Get("json")
	if !strings.HasPrefix(tag, "config") {
		t.Fatalf("RawConfig.Config should have json tag config, got %q", tag)
	}
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// small N: simple insertion sort
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

func getResourceByTypeName(t *testing.T, wantTypeName string) resource.Resource {
	t.Helper()

	p := provider.New("test")()
	for _, factory := range p.Resources(context.Background()) {
		res := factory()
		var resp resource.MetadataResponse
		res.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "peakhour"}, &resp)
		if resp.TypeName == wantTypeName {
			return res
		}
	}
	t.Fatalf("resource type %q not registered", wantTypeName)
	return nil
}
