package client

import "fmt"

func (c *Client) FlushRPCDNResources(domainName string, purge Purge) error {
	return c.Post(fmt.Sprintf("/api/v1/domains/%s/services/rp/cdn/resources/flush", domainName), purge, nil)
}

func (c *Client) FlushRPCDNWildcard(domainName string, purge Purge) error {
	return c.Post(fmt.Sprintf("/api/v1/domains/%s/services/rp/cdn/wildcard/flush", domainName), purge, nil)
}

func (c *Client) FlushRPCDNTags(domainName string, purge PurgeTags) error {
	return c.Post(fmt.Sprintf("/api/v1/domains/%s/services/rp/cdn/tag/flush", domainName), purge, nil)
}
