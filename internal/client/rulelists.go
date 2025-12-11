package client

import "fmt"

// ListRuleLists retrieves all rule lists for a domain
func (c *Client) ListRuleLists(domainName string) ([]RuleListSummary, error) {
	var result []RuleListSummary
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/lists", domainName), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetRuleList retrieves a specific rule list
func (c *Client) GetRuleList(domainName, listUUID string) (*RuleList, error) {
	var result RuleList
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/lists/%s", domainName, listUUID), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateRuleList creates a new rule list
func (c *Client) CreateRuleList(domainName string, list RuleListAdd) (*UUIDResult, error) {
	var result UUIDResult
	err := c.Post(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/lists", domainName), list, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateRuleList updates an existing rule list
func (c *Client) UpdateRuleList(domainName, listUUID string, list RuleListAdd) error {
	return c.Put(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/lists/%s", domainName, listUUID), list)
}

// DeleteRuleList deletes a rule list
func (c *Client) DeleteRuleList(domainName, listUUID string) error {
	return c.Delete(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/lists/%s", domainName, listUUID), nil)
}
