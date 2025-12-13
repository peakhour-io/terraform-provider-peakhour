package client

import (
	"fmt"
	"sort"
)

// CreateDomain creates a new domain
func (c *Client) CreateDomain(name string) (*Domain, error) {
	body := DomainAdd{Name: name}
	var result Domain
	err := c.Post("/api/v1/domains", body, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetDomain retrieves domain details
func (c *Client) GetDomain(name string) (*Domain, error) {
	var result Domain
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s", name), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteDomain deletes a domain
func (c *Client) DeleteDomain(name string) error {
	return c.Delete(fmt.Sprintf("/api/v1/domains/%s", name), nil)
}

// CreateDomainService creates a service for a domain
func (c *Client) CreateDomainService(domainName, serviceType string) error {
	body := DomainService{Type: serviceType}
	return c.Post(fmt.Sprintf("/api/v1/domains/%s/services", domainName), body, nil)
}

// DeleteDomainService deletes a service from a domain
func (c *Client) DeleteDomainService(domainName, serviceType string) error {
	return c.Delete(fmt.Sprintf("/api/v1/domains/%s/services/%s", domainName, serviceType), nil)
}

// GetDomainService checks if a service exists
func (c *Client) GetDomainService(domainName, serviceType string) error {
	var result interface{}
	return c.Get(fmt.Sprintf("/api/v1/domains/%s/services/%s", domainName, serviceType), &result)
}

// ListDomains returns the unique, sorted set of domain names visible to the API key.
// This includes both owned domains and granted domains.
func (c *Client) ListDomains() ([]string, error) {
	var result Domains
	if err := c.Get("/api/v1/domains", &result); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(result.Domains)+len(result.GrantedDomains))
	for _, d := range result.Domains {
		if d.Name == "" {
			continue
		}
		seen[d.Name] = struct{}{}
	}
	for _, d := range result.GrantedDomains {
		if d.Name == "" {
			continue
		}
		seen[d.Name] = struct{}{}
	}

	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}
