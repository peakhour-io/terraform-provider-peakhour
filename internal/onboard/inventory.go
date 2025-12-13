package onboard

import (
	"context"
	"fmt"
	"sort"

	"github.com/peakhour-io/terraform-provider-peakhour/internal/client"
)

type ImportTarget struct {
	TypeName  string
	Name      string
	ImportID  string
	DependsOn []string
}

type InventoryClient interface {
	GetDomainService(domainName, serviceType string) error
	GetReverseProxyConfig(domainName string) (*client.ReverseProxyConfig, error)
	GetTransformSettings(domainName string) (*client.TransformSettings, error)
	GetRateLimit(domainName string) (*client.RateLimit, error)
	ListRateLimitZones(domainName string) ([]client.RateLimitZone, error)
	GetOriginPools(domainName string) ([]client.OriginPool, error)
	ListRuleLists(domainName string) ([]client.RuleListSummary, error)
	ListRulesInPhase(domainName, phase string) ([]client.RulePhaseSummary, error)
	ListImageTransformPresets(domainName string) ([]client.ImageTransformPreset, error)
	ListBulkRedirectLists(domainName string) ([]client.BulkRedirectListSummary, error)
	ListBulkRedirectEntries(domainName, listUUID string) ([]client.BulkRedirectEntry, error)
}

func CollectDomainInventory(ctx context.Context, c InventoryClient, domain string) ([]ImportTarget, error) {
	_ = ctx

	targets := []ImportTarget{
		{TypeName: "peakhour_domain", Name: "domain", ImportID: domain},
	}

	if err := c.GetDomainService(domain, "rp"); err != nil {
		if !client.IsNotFoundError(err) {
			return nil, err
		}
	} else {
		targets = append(targets, ImportTarget{TypeName: "peakhour_reverse_proxy_service", Name: "rp", ImportID: domain})
	}

	if _, err := c.GetReverseProxyConfig(domain); err != nil {
		if !client.IsNotFoundError(err) {
			return nil, err
		}
	} else {
		targets = append(targets, ImportTarget{TypeName: "peakhour_reverse_proxy_config", Name: "config", ImportID: domain})
	}

	if _, err := c.GetTransformSettings(domain); err != nil {
		if !client.IsNotFoundError(err) {
			return nil, err
		}
	} else {
		targets = append(targets, ImportTarget{TypeName: "peakhour_transform_settings", Name: "transforms", ImportID: domain})
	}

	if _, err := c.GetRateLimit(domain); err != nil {
		if !client.IsNotFoundError(err) {
			return nil, err
		}
	} else {
		targets = append(targets, ImportTarget{TypeName: "peakhour_rate_limit_global", Name: "global", ImportID: domain})
	}

	if zones, err := c.ListRateLimitZones(domain); err != nil {
		if !client.IsNotFoundError(err) {
			return nil, err
		}
	} else {
		for _, z := range zones {
			if z.Name == "" {
				continue
			}
			targets = append(targets, ImportTarget{
				TypeName: "peakhour_rate_limit_zone",
				Name:     z.Name,
				ImportID: domain + "/" + z.Name,
			})
		}
	}

	if pools, err := c.GetOriginPools(domain); err != nil {
		if !client.IsNotFoundError(err) {
			return nil, err
		}
	} else {
		for _, p := range pools {
			if p.Tag == "" {
				continue
			}
			targets = append(targets, ImportTarget{
				TypeName: "peakhour_origin_pool",
				Name:     p.Tag,
				ImportID: domain + "/origins/" + p.Tag,
			})
		}
	}

	if lists, err := c.ListRuleLists(domain); err != nil {
		if !client.IsNotFoundError(err) {
			return nil, err
		}
	} else {
		for _, l := range lists {
			if l.UUID == "" {
				continue
			}
			targets = append(targets, ImportTarget{
				TypeName: "peakhour_rule_list",
				Name:     l.UUID,
				ImportID: domain + "/" + l.UUID,
			})
		}
	}

	for _, phase := range []string{
		"request_rewrite",
		"url_config",
		"firewall",
		"rate_limit_request",
		"rate_limit_request_late",
		"rate_limit_response",
		"request_headers",
		"response_headers",
		"load_balance",
		"bulk_redirect",
	} {
		rules, err := c.ListRulesInPhase(domain, phase)
		if err != nil {
			if client.IsNotFoundError(err) {
				continue
			}
			return nil, err
		}
		for _, r := range rules {
			if r.UUID == "" {
				continue
			}
			targets = append(targets, ImportTarget{
				TypeName: "peakhour_rule",
				Name:     fmt.Sprintf("%s_%s", phase, r.UUID),
				ImportID: fmt.Sprintf("%s/%s/%s", domain, phase, r.UUID),
			})
		}
	}

	if presets, err := c.ListImageTransformPresets(domain); err != nil {
		if !client.IsNotFoundError(err) {
			return nil, err
		}
	} else {
		for _, p := range presets {
			if p.UUID == "" {
				continue
			}
			targets = append(targets, ImportTarget{
				TypeName: "peakhour_image_transform",
				Name:     p.UUID,
				ImportID: domain + "/" + p.UUID,
			})
		}
	}

	if lists, err := c.ListBulkRedirectLists(domain); err != nil {
		if !client.IsNotFoundError(err) {
			return nil, err
		}
	} else {
		for _, l := range lists {
			if l.UUID == "" {
				continue
			}
			targets = append(targets, ImportTarget{
				TypeName: "peakhour_bulk_redirect_list",
				Name:     l.UUID,
				ImportID: fmt.Sprintf("%s/bulk_redirects/%s", domain, l.UUID),
			})

			entries, err := c.ListBulkRedirectEntries(domain, l.UUID)
			if err != nil {
				if client.IsNotFoundError(err) {
					continue
				}
				return nil, err
			}
			for _, e := range entries {
				if e.ID == "" {
					continue
				}
				targets = append(targets, ImportTarget{
					TypeName: "peakhour_bulk_redirect_entry",
					Name:     fmt.Sprintf("%s_%s", l.UUID, e.ID),
					ImportID: fmt.Sprintf("%s/bulk_redirects/%s/entries/%s", domain, l.UUID, e.ID),
				})
			}
		}
	}

	sort.SliceStable(targets, func(i, j int) bool {
		if targets[i].TypeName != targets[j].TypeName {
			return targets[i].TypeName < targets[j].TypeName
		}
		return targets[i].Name < targets[j].Name
	})

	return targets, nil
}
