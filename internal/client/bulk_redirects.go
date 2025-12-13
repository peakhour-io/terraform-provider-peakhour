package client

import "fmt"

// ListBulkRedirectLists retrieves all bulk redirect lists for a domain.
func (c *Client) ListBulkRedirectLists(domainName string) ([]BulkRedirectListSummary, error) {
	var result []BulkRedirectListSummary
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/bulk_redirects", domainName), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetBulkRedirectList retrieves a single bulk redirect list by UUID.
func (c *Client) GetBulkRedirectList(domainName, listUUID string) (*BulkRedirectList, error) {
	var result BulkRedirectList
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/bulk_redirects/%s", domainName, listUUID), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateBulkRedirectList creates a new bulk redirect list.
func (c *Client) CreateBulkRedirectList(domainName string, create BulkRedirectListCreate) (*UUIDResult, error) {
	var result UUIDResult
	if err := c.Post(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/bulk_redirects", domainName), create, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateBulkRedirectList updates a bulk redirect list by UUID.
func (c *Client) UpdateBulkRedirectList(domainName, listUUID string, update BulkRedirectListUpdate) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/bulk_redirects/%s", domainName, listUUID), update)
}

// DeleteBulkRedirectList deletes a bulk redirect list by UUID.
func (c *Client) DeleteBulkRedirectList(domainName, listUUID string) error {
	return c.Delete(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/bulk_redirects/%s", domainName, listUUID), nil)
}

// ListBulkRedirectEntries retrieves all entries for a bulk redirect list.
func (c *Client) ListBulkRedirectEntries(domainName, listUUID string) ([]BulkRedirectEntry, error) {
	var result []BulkRedirectEntry
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/bulk_redirects/%s/entries", domainName, listUUID), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateBulkRedirectEntry creates a new entry for a bulk redirect list.
func (c *Client) CreateBulkRedirectEntry(domainName, listUUID string, create BulkRedirectEntryCreate) (*BulkRedirectEntry, error) {
	var result BulkRedirectEntry
	if err := c.Post(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/bulk_redirects/%s/entries", domainName, listUUID), create, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// BulkCreateBulkRedirectEntries creates multiple entries in a single request.
func (c *Client) BulkCreateBulkRedirectEntries(domainName, listUUID string, create BulkRedirectEntryBulkCreate) error {
	return c.Put(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/bulk_redirects/%s/entries", domainName, listUUID), create)
}

// UpdateBulkRedirectEntry updates an existing entry in a bulk redirect list.
func (c *Client) UpdateBulkRedirectEntry(domainName, listUUID, entryID string, update BulkRedirectEntryUpdate) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/bulk_redirects/%s/entries/%s", domainName, listUUID, entryID), update)
}

// DeleteBulkRedirectEntry deletes an entry from a bulk redirect list.
func (c *Client) DeleteBulkRedirectEntry(domainName, listUUID, entryID string) error {
	return c.Delete(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/bulk_redirects/%s/entries/%s", domainName, listUUID, entryID), nil)
}
