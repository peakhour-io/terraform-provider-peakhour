package client

import "fmt"

func (c *Client) GetRPCDNCacheConfig(domainName string) (*CacheConfig, error) {
	var result CacheConfig
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/cdn", domainName), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateRPCDNCacheConfig patches the CDN cache config for a domain.
//
// The API is PATCH-based and supports null values for clearing. Pass a map so callers
// can distinguish between omitting a field and explicitly setting it to null.
func (c *Client) UpdateRPCDNCacheConfig(domainName string, update map[string]any) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp/cdn", domainName), update)
}
