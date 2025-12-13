package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"peakhour": providerserver.NewProtocol6WithError(New("testacc")()),
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping acceptance tests")
	}

	// terraform-plugin-testing defaults to serving/reattaching providers under
	// the "hashicorp" namespace. This provider uses the "peakhour-io" namespace,
	// so ensure the testing harness generates matching provider addresses.
	t.Setenv("TF_ACC_PROVIDER_NAMESPACE", "peakhour-io")

	if os.Getenv("PEAKHOUR_API_KEY") == "" {
		t.Fatal("PEAKHOUR_API_KEY must be set for acceptance tests")
	}
}

func testAccEnv(t *testing.T, key string) string {
	t.Helper()

	value := os.Getenv(key)
	if value == "" {
		t.Fatalf("%s must be set for acceptance tests", key)
	}
	return value
}
