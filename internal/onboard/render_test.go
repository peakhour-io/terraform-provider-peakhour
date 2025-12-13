package onboard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRenderProviderTF_Golden(t *testing.T) {
	wantPath := filepath.Join("testdata", "provider.tf.golden")
	wantBytes, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read %s: %v", wantPath, err)
	}

	got := RenderProviderTF()
	if diff := cmp.Diff(string(wantBytes), got); diff != "" {
		t.Fatalf("RenderProviderTF mismatch (-want +got):\n%s", diff)
	}
}

func TestRenderImportsTF_Golden(t *testing.T) {
	wantPath := filepath.Join("testdata", "imports.tf.golden")
	wantBytes, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read %s: %v", wantPath, err)
	}

	targets := []ImportTarget{
		{TypeName: "peakhour_rule_list", Name: "foo bar", ImportID: "example.com/list-2"},
		{TypeName: "peakhour_domain", Name: "domain", ImportID: "example.com"},
		{TypeName: "peakhour_origin_pool", Name: "123Pool", ImportID: "example.com/origins/123Pool"},
		{TypeName: "peakhour_rule_list", Name: "Foo-Bar", ImportID: "example.com/list-1"},
		{TypeName: "peakhour_rate_limit_zone", Name: "Zone A", ImportID: "example.com/zone-a"},
	}

	got := RenderImportsTF(targets)
	if diff := cmp.Diff(string(wantBytes), got); diff != "" {
		t.Fatalf("RenderImportsTF mismatch (-want +got):\n%s", diff)
	}
}

