package client

import "fmt"

func (c *Client) ListRPWAFRuleGroups(domainName string, ruleset string) ([]WAFRuleGroup, error) {
	var groups []WAFRuleGroup
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/waf/ruleset/%s", domainName, ruleset), &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

func (c *Client) SetRPWAFRuleGroupEnabled(domainName string, ruleset string, rulegroup string, enabled bool) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp/waf/ruleset/%s/rulegroup/%s", domainName, ruleset, rulegroup), WAFToggle{State: enabled})
}
