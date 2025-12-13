package client

import "fmt"

func (c *Client) GetRPWAFOptions(domainName string) (*WAFOptions, error) {
	var result WAFOptions
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/waf", domainName), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateRPWAFOptions patches WAF options for a domain.
//
// The API is PATCH-based and supports null values for clearing. Pass a map so callers
// can distinguish between omitting a field and explicitly setting it to null.
func (c *Client) UpdateRPWAFOptions(domainName string, update map[string]any) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp/waf", domainName), update)
}

func (c *Client) GetRPWAFOWASPSettings(domainName string) (map[string]any, error) {
	var result map[string]any
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/waf/owasp", domainName), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateRPWAFOWASPSettings patches OWASP settings for a domain.
//
// The API is PATCH-based and supports null values for clearing. Pass a map so callers
// can distinguish between omitting a field and explicitly setting it to null.
func (c *Client) UpdateRPWAFOWASPSettings(domainName string, update map[string]any) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp/waf/owasp", domainName), update)
}
