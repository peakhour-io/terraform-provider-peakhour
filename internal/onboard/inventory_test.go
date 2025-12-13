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
	rpSettings         *client.ServiceSettings
	rpSSLConfig        *client.SSLConfig
	rpSSLCertificate   *client.SSLCertificate
	transformSettings  *client.TransformSettings
	acmeSettings       *client.AcmeSettings
	acmeCertificate    *client.AcmeCertificate
	rateLimit          *client.RateLimit

	rateLimitZones    []client.RateLimitZone
	originPools       []client.OriginPool
	originConfig      *client.OriginConfig
	cdnCacheConfig    *client.CacheConfig
	botsConfig        *client.BotConfig
	threatAccessRules []client.AccessListRule
	threatBlockLists  []client.Blocklist
	firewallSettings  *client.FirewallSettings
	firewallErrorPage *client.FirewallErrorPage
	luaOptions        *client.LuaOptions
	wafOptions        *client.WAFOptions
	wafOWASPSettings  map[string]any
	wafCustomRules    []client.WAFCustomRule
	ruleLists         []client.RuleListSummary
	rulesByPhase      map[string][]client.RulePhaseSummary

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

func (f *fakeInventoryClient) GetRPSettings(domainName string) (*client.ServiceSettings, error) {
	return f.rpSettings, nil
}

func (f *fakeInventoryClient) GetRPSSLConfig(domainName string) (*client.SSLConfig, error) {
	return f.rpSSLConfig, nil
}

func (f *fakeInventoryClient) GetRPSSLCertificate(domainName string) (*client.SSLCertificate, error) {
	return f.rpSSLCertificate, nil
}

func (f *fakeInventoryClient) GetTransformSettings(domainName string) (*client.TransformSettings, error) {
	return f.transformSettings, nil
}

func (f *fakeInventoryClient) GetAcmeSettings(domainName string) (*client.AcmeSettings, error) {
	return f.acmeSettings, nil
}

func (f *fakeInventoryClient) GetAcmeCertificate(domainName string) (*client.AcmeCertificate, error) {
	return f.acmeCertificate, nil
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

func (f *fakeInventoryClient) GetRPOriginConfig(domainName string) (*client.OriginConfig, error) {
	return f.originConfig, nil
}

func (f *fakeInventoryClient) GetRPCDNCacheConfig(domainName string) (*client.CacheConfig, error) {
	return f.cdnCacheConfig, nil
}

func (f *fakeInventoryClient) GetRPBotsConfig(domainName string) (*client.BotConfig, error) {
	return f.botsConfig, nil
}

func (f *fakeInventoryClient) ListThreatAccessListRules(domainName string) ([]client.AccessListRule, error) {
	return f.threatAccessRules, nil
}

func (f *fakeInventoryClient) ListThreatBlockLists(domainName string) ([]client.Blocklist, error) {
	return f.threatBlockLists, nil
}

func (f *fakeInventoryClient) GetRPFirewallSettings(domainName string) (*client.FirewallSettings, error) {
	return f.firewallSettings, nil
}

func (f *fakeInventoryClient) GetRPFirewallErrorPage(domainName string) (*client.FirewallErrorPage, error) {
	return f.firewallErrorPage, nil
}

func (f *fakeInventoryClient) GetRPLuaOptions(domainName string) (*client.LuaOptions, error) {
	return f.luaOptions, nil
}

func (f *fakeInventoryClient) GetRPWAFOptions(domainName string) (*client.WAFOptions, error) {
	return f.wafOptions, nil
}

func (f *fakeInventoryClient) GetRPWAFOWASPSettings(domainName string) (map[string]any, error) {
	return f.wafOWASPSettings, nil
}

func (f *fakeInventoryClient) ListRPWAFCustomRules(domainName string) ([]client.WAFCustomRule, error) {
	return f.wafCustomRules, nil
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
		rpSettings:         &client.ServiceSettings{},
		rpSSLConfig:        &client.SSLConfig{},
		rpSSLCertificate:   &client.SSLCertificate{},
		transformSettings:  &client.TransformSettings{},
		acmeSettings:       &client.AcmeSettings{},
		acmeCertificate:    &client.AcmeCertificate{},
		rateLimit:          &client.RateLimit{},
		rateLimitZones: []client.RateLimitZone{
			{Name: "zone-b"},
			{Name: "zone-a"},
		},
		originPools: []client.OriginPool{
			{Tag: "pool-b"},
			{Tag: "pool-a"},
		},
		originConfig:   &client.OriginConfig{},
		cdnCacheConfig: &client.CacheConfig{},
		botsConfig:     &client.BotConfig{},
		threatAccessRules: []client.AccessListRule{
			{UUID: "al-2"},
			{UUID: "al-1"},
		},
		threatBlockLists: []client.Blocklist{
			{Name: "tor", Enabled: true},
		},
		firewallSettings:  &client.FirewallSettings{},
		firewallErrorPage: &client.FirewallErrorPage{ErrorPage: true},
		luaOptions:        &client.LuaOptions{},
		wafOptions:        &client.WAFOptions{},
		wafOWASPSettings:  map[string]any{},
		wafCustomRules: []client.WAFCustomRule{
			{UUID: "wafcr-2"},
			{UUID: "wafcr-1"},
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
		{TypeName: "peakhour_acme_certificate", Name: "acme_certificate", ImportID: "example.com"},
		{TypeName: "peakhour_acme_settings", Name: "acme", ImportID: "example.com"},
		{TypeName: "peakhour_bulk_redirect_entry", Name: "br-1_entry-1", ImportID: "example.com/bulk_redirects/br-1/entries/entry-1"},
		{TypeName: "peakhour_bulk_redirect_entry", Name: "br-1_entry-2", ImportID: "example.com/bulk_redirects/br-1/entries/entry-2"},
		{TypeName: "peakhour_bulk_redirect_list", Name: "br-1", ImportID: "example.com/bulk_redirects/br-1"},
		{TypeName: "peakhour_domain", Name: "domain", ImportID: "example.com"},
		{TypeName: "peakhour_image_transform", Name: "img-1", ImportID: "example.com/img-1"},
		{TypeName: "peakhour_image_transform", Name: "img-2", ImportID: "example.com/img-2"},
		{TypeName: "peakhour_origin_pool", Name: "pool-a", ImportID: "example.com/origins/pool-a"},
		{TypeName: "peakhour_origin_pool", Name: "pool-b", ImportID: "example.com/origins/pool-b"},
		{TypeName: "peakhour_rate_limit_global", Name: "global", ImportID: "example.com"},
		{TypeName: "peakhour_rate_limit_settings", Name: "settings", ImportID: "example.com"},
		{TypeName: "peakhour_rate_limit_zone", Name: "zone-a", ImportID: "example.com/zone-a"},
		{TypeName: "peakhour_rate_limit_zone", Name: "zone-b", ImportID: "example.com/zone-b"},
		{TypeName: "peakhour_reverse_proxy_config", Name: "config", ImportID: "example.com"},
		{TypeName: "peakhour_reverse_proxy_service", Name: "rp", ImportID: "example.com"},
		{TypeName: "peakhour_rp_bots", Name: "bots", ImportID: "example.com"},
		{TypeName: "peakhour_rp_cdn_cache", Name: "cache", ImportID: "example.com"},
		{TypeName: "peakhour_rp_firewall_error_page", Name: "error_page", ImportID: "example.com"},
		{TypeName: "peakhour_rp_firewall_settings", Name: "firewall", ImportID: "example.com"},
		{TypeName: "peakhour_rp_lua_options", Name: "lua", ImportID: "example.com"},
		{TypeName: "peakhour_rp_origin_config", Name: "origin", ImportID: "example.com"},
		{TypeName: "peakhour_rp_settings", Name: "settings", ImportID: "example.com"},
		{TypeName: "peakhour_rp_ssl_certificate", Name: "ssl_certificate", ImportID: "example.com"},
		{TypeName: "peakhour_rp_ssl_config", Name: "ssl", ImportID: "example.com"},
		{TypeName: "peakhour_rp_threat_access_list_rule", Name: "al-1", ImportID: "example.com/access_list/al-1"},
		{TypeName: "peakhour_rp_threat_access_list_rule", Name: "al-2", ImportID: "example.com/access_list/al-2"},
		{TypeName: "peakhour_rp_threat_block_list", Name: "threat_block_list", ImportID: "example.com"},
		{TypeName: "peakhour_rp_waf_custom_rule", Name: "wafcr-1", ImportID: "example.com/customrule/wafcr-1"},
		{TypeName: "peakhour_rp_waf_custom_rule", Name: "wafcr-2", ImportID: "example.com/customrule/wafcr-2"},
		{TypeName: "peakhour_rp_waf_options", Name: "waf", ImportID: "example.com"},
		{TypeName: "peakhour_rp_waf_owasp_settings", Name: "waf_owasp", ImportID: "example.com"},
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
