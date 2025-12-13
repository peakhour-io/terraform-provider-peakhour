package client

import "fmt"

// GetRateLimit retrieves the full rate limit configuration (global + zones) for a domain.
func (c *Client) GetRateLimit(domainName string) (*RateLimit, error) {
	var result RateLimit
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/rate_limit", domainName), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateRateLimitSettings updates the rate limit mode configuration for a domain.
func (c *Client) UpdateRateLimitSettings(domainName string, settings RateLimitSettings) error {
	return c.Post(fmt.Sprintf("/api/v1/domains/%s/services/rp/rate_limit", domainName), settings, nil)
}

// UpdateRateLimitGlobal updates the global rate limiting configuration for a domain.
func (c *Client) UpdateRateLimitGlobal(domainName string, global RateLimitGlobal) error {
	return c.Post(fmt.Sprintf("/api/v1/domains/%s/services/rp/rate_limit/global", domainName), global, nil)
}

// ListRateLimitZones retrieves all rate limit zones for a domain
func (c *Client) ListRateLimitZones(domainName string) ([]RateLimitZone, error) {
	var result []RateLimitZone
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/rate_limit/zones", domainName), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetRateLimitZone retrieves a specific rate limit zone
func (c *Client) GetRateLimitZone(domainName, zoneName string) (*RateLimitZone, error) {
	var result RateLimitZone
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/rate_limit/zones/%s", domainName, zoneName), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateRateLimitZone creates a new rate limit zone
func (c *Client) CreateRateLimitZone(domainName string, zone RateLimitZone) (*RateLimitZone, error) {
	var result RateLimitZone
	err := c.Post(fmt.Sprintf("/api/v1/domains/%s/services/rp/rate_limit/zones", domainName), zone, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateRateLimitZone updates an existing rate limit zone
func (c *Client) UpdateRateLimitZone(domainName, zoneName string, zone RateLimitZone) error {
	return c.Put(fmt.Sprintf("/api/v1/domains/%s/services/rp/rate_limit/zones/%s", domainName, zoneName), zone)
}

// DeleteRateLimitZone deletes a rate limit zone
func (c *Client) DeleteRateLimitZone(domainName, zoneName string) error {
	return c.Delete(fmt.Sprintf("/api/v1/domains/%s/services/rp/rate_limit/zones/%s", domainName, zoneName), nil)
}
