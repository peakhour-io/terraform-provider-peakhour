package client

import "fmt"

func (c *Client) GetRPFirewallSettings(domainName string) (*FirewallSettings, error) {
	var result FirewallSettings
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/firewall", domainName), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateRPFirewallSettings updates firewall settings for a domain.
//
// The API is POST-based and supports null values for clearing. Pass a map so callers
// can distinguish between omitting a field and explicitly setting it to null.
func (c *Client) UpdateRPFirewallSettings(domainName string, update map[string]any) error {
	return c.Post(fmt.Sprintf("/api/v1/domains/%s/services/rp/firewall", domainName), update, nil)
}

func (c *Client) GetRPFirewallErrorPage(domainName string) (*FirewallErrorPage, error) {
	var result FirewallErrorPage
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/firewall/error_page", domainName), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateRPFirewallErrorPage(domainName string, update map[string]any) error {
	return c.Put(fmt.Sprintf("/api/v1/domains/%s/services/rp/firewall/error_page", domainName), update)
}
