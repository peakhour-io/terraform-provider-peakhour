package client

import "fmt"

func (c *Client) ListThreatAccessListRules(domainName string) ([]AccessListRule, error) {
	var rules []AccessListRule
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/threats/access_list", domainName), &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (c *Client) CreateThreatAccessListRule(domainName string, rule AccessListRuleAdd) (*AccessListRule, error) {
	var created AccessListRule
	if err := c.PutAndDecode(fmt.Sprintf("/api/v1/domains/%s/services/rp/threats/access_list", domainName), rule, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) GetThreatAccessListRule(domainName string, ruleUUID string) (*AccessListRule, error) {
	var rule AccessListRule
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/threats/access_list/%s", domainName, ruleUUID), &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (c *Client) UpdateThreatAccessListRule(domainName string, ruleUUID string, update AccessListRuleUpdate) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp/threats/access_list/%s", domainName, ruleUUID), update)
}

func (c *Client) DeleteThreatAccessListRule(domainName string, ruleUUID string) error {
	return c.Delete(fmt.Sprintf("/api/v1/domains/%s/services/rp/threats/access_list/%s", domainName, ruleUUID), nil)
}

func (c *Client) ListThreatBlockLists(domainName string) ([]Blocklist, error) {
	var lists []Blocklist
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/threats/block_list", domainName), &lists); err != nil {
		return nil, err
	}
	return lists, nil
}

func (c *Client) SetThreatBlockLists(domainName string, set BlocklistsSet) error {
	return c.Post(fmt.Sprintf("/api/v1/domains/%s/services/rp/threats/block_list", domainName), set, nil)
}
