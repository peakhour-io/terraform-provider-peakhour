package client

import "fmt"

// GetReverseProxyConfig retrieves reverse proxy configuration
func (c *Client) GetReverseProxyConfig(domainName string) (*ReverseProxyConfig, error) {
	var result ReverseProxyConfig
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp", domainName), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateReverseProxyConfig updates reverse proxy configuration
func (c *Client) UpdateReverseProxyConfig(domainName string, config ReverseProxyConfig) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp", domainName), config)
}
