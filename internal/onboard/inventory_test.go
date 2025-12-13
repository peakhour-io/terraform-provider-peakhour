package onboard

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

type fakeInventoryClient struct {
	domainServiceErr error

	reverseProxyConfig *client.ReverseProxyConfig
	transformSettings  *client.TransformSettings
	rateLimit          *client.RateLimit

	rateLimitZones []client.RateLimitZone
	originPools    []client.OriginPool
	ruleLists      []client.RuleListSummary
	rulesByPhase   map[string][]client.RulePhaseSummary

	imageTransforms []client.ImageTransformPreset

	bulkRedirectLists   []client.BulkRedirectListSummary
	bulkRedirectEntries map[string][]client.BulkRedirectEntry
}

func (f *fakeInventoryClient) GetDomainService(domainName, serviceType string) error {
	return f.domainServiceErr
}

func (f *fakeInventoryClient) GetReverseProxyConfig(domainName string) (*client.ReverseProxyConfig, error) {
	return f.reverseProxyConfig, nil
}

func (f *fakeInventoryClient) GetTransformSettings(domainName string) (*client.TransformSettings, error) {
	return f.transformSettings, nil
}

func (f *fakeInventoryClient) GetRateLimit(domainName string) (*client.RateLimit, error) {
	return f.rateLimit, nil
}

func (f *fakeInventoryClient) ListRateLimitZones(domainName string) ([]client.RateLimitZone, error) {
	return f.rateLimitZones, nil
}

func (f *fakeInventoryClient) GetOriginPools(domainName string) ([]client.OriginPool, error) {
	return f.originPools, nil
}

func (f *fakeInventoryClient) ListRuleLists(domainName string) ([]client.RuleListSummary, error) {
	return f.ruleLists, nil
}

func (f *fakeInventoryClient) ListRulesInPhase(domainName, phase string) ([]client.RulePhaseSummary, error) {
	return f.rulesByPhase[phase], nil
}

func (f *fakeInventoryClient) ListImageTransformPresets(domainName string) ([]client.ImageTransformPreset, error) {
	return f.imageTransforms, nil
}

func (f *fakeInventoryClient) ListBulkRedirectLists(domainName string) ([]client.BulkRedirectListSummary, error) {
	return f.bulkRedirectLists, nil
}

func (f *fakeInventoryClient) ListBulkRedirectEntries(domainName, listUUID string) ([]client.BulkRedirectEntry, error) {
	return f.bulkRedirectEntries[listUUID], nil
}

func TestCollectDomainInventory_Basic(t *testing.T) {
	fake := &fakeInventoryClient{
		reverseProxyConfig: &client.ReverseProxyConfig{},
		transformSettings:  &client.TransformSettings{},
		rateLimit:          &client.RateLimit{},
		rateLimitZones: []client.RateLimitZone{
			{Name: "zone-b"},
			{Name: "zone-a"},
		},
		originPools: []client.OriginPool{
			{Tag: "pool-b"},
			{Tag: "pool-a"},
		},
		ruleLists: []client.RuleListSummary{
			{UUID: "list-2", Name: "My List 2", Type: "ips"},
			{UUID: "list-1", Name: "My List 1", Type: "ips"},
		},
		rulesByPhase: map[string][]client.RulePhaseSummary{
			"firewall": {
				{UUID: "rule-2", Phase: "firewall", Pos: 2, Name: "Block 2"},
				{UUID: "rule-1", Phase: "firewall", Pos: 1, Name: "Block 1"},
			},
		},
		imageTransforms: []client.ImageTransformPreset{
			{UUID: "img-2", Name: "Preset 2"},
			{UUID: "img-1", Name: "Preset 1"},
		},
		bulkRedirectLists: []client.BulkRedirectListSummary{
			{UUID: "br-1", Name: "Redirects"},
		},
		bulkRedirectEntries: map[string][]client.BulkRedirectEntry{
			"br-1": {
				{ID: "entry-2"},
				{ID: "entry-1"},
			},
		},
	}

	got, err := CollectDomainInventory(context.Background(), fake, "example.com")
	if err != nil {
		t.Fatalf("CollectDomainInventory returned error: %v", err)
	}

	want := []ImportTarget{
		{TypeName: "peakhour_bulk_redirect_entry", Name: "br-1_entry-1", ImportID: "example.com/bulk_redirects/br-1/entries/entry-1"},
		{TypeName: "peakhour_bulk_redirect_entry", Name: "br-1_entry-2", ImportID: "example.com/bulk_redirects/br-1/entries/entry-2"},
		{TypeName: "peakhour_bulk_redirect_list", Name: "br-1", ImportID: "example.com/bulk_redirects/br-1"},
		{TypeName: "peakhour_domain", Name: "domain", ImportID: "example.com"},
		{TypeName: "peakhour_image_transform", Name: "img-1", ImportID: "example.com/img-1"},
		{TypeName: "peakhour_image_transform", Name: "img-2", ImportID: "example.com/img-2"},
		{TypeName: "peakhour_origin_pool", Name: "pool-a", ImportID: "example.com/origins/pool-a"},
		{TypeName: "peakhour_origin_pool", Name: "pool-b", ImportID: "example.com/origins/pool-b"},
		{TypeName: "peakhour_rate_limit_global", Name: "global", ImportID: "example.com"},
		{TypeName: "peakhour_rate_limit_zone", Name: "zone-a", ImportID: "example.com/zone-a"},
		{TypeName: "peakhour_rate_limit_zone", Name: "zone-b", ImportID: "example.com/zone-b"},
		{TypeName: "peakhour_reverse_proxy_config", Name: "config", ImportID: "example.com"},
		{TypeName: "peakhour_reverse_proxy_service", Name: "rp", ImportID: "example.com"},
		{TypeName: "peakhour_rule", Name: "firewall_rule-1", ImportID: "example.com/firewall/rule-1"},
		{TypeName: "peakhour_rule", Name: "firewall_rule-2", ImportID: "example.com/firewall/rule-2"},
		{TypeName: "peakhour_rule_list", Name: "list-1", ImportID: "example.com/list-1"},
		{TypeName: "peakhour_rule_list", Name: "list-2", ImportID: "example.com/list-2"},
		{TypeName: "peakhour_transform_settings", Name: "transforms", ImportID: "example.com"},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("CollectDomainInventory mismatch (-want +got):\n%s", diff)
	}
}
