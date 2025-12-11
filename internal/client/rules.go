package client

import "fmt"

// ListRules retrieves all rules for a domain
func (c *Client) ListRules(domainName string) ([]RulePhaseSummary, error) {
	var result []RulePhaseSummary
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules", domainName), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ListRulesInPhase retrieves rules in a specific phase
func (c *Client) ListRulesInPhase(domainName, phase string) ([]RulePhaseSummary, error) {
	var result []RulePhaseSummary
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/phases/%s", domainName, phase), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetRule retrieves a specific rule by UUID
func (c *Client) GetRule(domainName, phase, ruleUUID string) (*RulePhase, error) {
	var result RulePhase
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/phases/%s/rule/%s", domainName, phase, ruleUUID), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateRule creates a new rule in a phase
func (c *Client) CreateRule(domainName string, rule RulePhaseAdd) (*UUIDResult, error) {
	var result UUIDResult
	err := c.Post(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/phases/%s", domainName, rule.Phase), rule, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateRule updates an existing rule
func (c *Client) UpdateRule(domainName, phase, ruleUUID string, update RulePhaseUpdate) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/phases/%s/rule/%s", domainName, phase, ruleUUID), update)
}

// DeleteRule deletes a rule
func (c *Client) DeleteRule(domainName, phase, ruleUUID string) error {
	return c.Delete(fmt.Sprintf("/api/v1/domains/%s/services/rp/rules/phases/%s/rule/%s", domainName, phase, ruleUUID), nil)
}
