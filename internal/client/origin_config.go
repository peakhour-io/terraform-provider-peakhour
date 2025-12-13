package client

import "fmt"

func (c *Client) GetRPOriginConfig(domainName string) (*OriginConfig, error) {
	var result OriginConfig
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/origin", domainName), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateRPOriginConfig updates the RP origin config for a domain.
//
// The API is POST-based and accepts null values for clearing. A map is used so
// callers can distinguish between omitting a field and explicitly setting it to null.
func (c *Client) UpdateRPOriginConfig(domainName string, update map[string]any) error {
	return c.Post(fmt.Sprintf("/api/v1/domains/%s/services/rp/origin", domainName), update, nil)
}
