package client

import "fmt"

func (c *Client) GetRPBotsConfig(domainName string) (*BotConfig, error) {
	var result BotConfig
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/bots", domainName), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateRPBotsConfig patches the bots config for a domain.
//
// The API is PATCH-based and supports null values for clearing. Pass a map so callers
// can distinguish between omitting a field and explicitly setting it to null.
func (c *Client) UpdateRPBotsConfig(domainName string, update map[string]any) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp/bots", domainName), update)
}
