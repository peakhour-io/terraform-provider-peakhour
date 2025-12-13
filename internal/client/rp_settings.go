package client

import "fmt"

func (c *Client) GetRPSettings(domainName string) (*ServiceSettings, error) {
	var result ServiceSettings
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/settings", domainName), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateRPSettings patches the RP service settings for a domain.
//
// The API is PATCH-based and supports null values for clearing. Pass a map so callers
// can distinguish between omitting a field and explicitly setting it to null.
func (c *Client) UpdateRPSettings(domainName string, update map[string]any) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp/settings", domainName), update)
}
