package client

import "fmt"

func (c *Client) ListRPWAFCustomRules(domainName string) ([]WAFCustomRule, error) {
	var rules []WAFCustomRule
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/waf/customrule", domainName), &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (c *Client) CreateRPWAFCustomRule(domainName string, rule map[string]any) (*WAFCustomRule, error) {
	var created WAFCustomRule
	if err := c.PutAndDecode(fmt.Sprintf("/api/v1/domains/%s/services/rp/waf/customrule", domainName), rule, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateRPWAFCustomRule(domainName string, ruleUUID string, rule map[string]any) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp/waf/customrule/%s", domainName, ruleUUID), rule)
}

func (c *Client) DeleteRPWAFCustomRule(domainName string, ruleUUID string) error {
	return c.Delete(fmt.Sprintf("/api/v1/domains/%s/services/rp/waf/customrule/%s", domainName, ruleUUID), nil)
}

func (c *Client) EnableRPWAFCustomRule(domainName string, ruleUUID string) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp/waf/customrule/%s/enable", domainName, ruleUUID), nil)
}

func (c *Client) ReorderRPWAFCustomRules(domainName string, order []string) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp/waf/customrule", domainName), WAFCustomRuleReorder{Order: order})
}
