package client

import "fmt"

// GetOriginPools retrieves all origin pools for a domain
func (c *Client) GetOriginPools(domainName string) ([]OriginPool, error) {
	var result []OriginPool
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/origins", domainName), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetOriginPool retrieves a specific origin pool by tag
func (c *Client) GetOriginPool(domainName, tag string) (*OriginPool, error) {
	pools, err := c.GetOriginPools(domainName)
	if err != nil {
		return nil, err
	}

	for _, pool := range pools {
		if pool.Tag == tag {
			return &pool, nil
		}
	}

	return nil, fmt.Errorf("origin pool with tag '%s' not found", tag)
}

// CreateOriginPool creates a new origin pool
func (c *Client) CreateOriginPool(domainName string, pool OriginPool) error {
	return c.Post(fmt.Sprintf("/api/v1/domains/%s/origins", domainName), pool, nil)
}

// UpdateOriginPool updates an existing origin pool
func (c *Client) UpdateOriginPool(domainName string, pool OriginPool) error {
	return c.Put(fmt.Sprintf("/api/v1/domains/%s/origins", domainName), pool)
}

// DeleteOriginPool deletes an origin pool
func (c *Client) DeleteOriginPool(domainName, tag string) error {
	body := OriginPool{Tag: tag}
	return c.Delete(fmt.Sprintf("/api/v1/domains/%s/origins", domainName), body)
}
