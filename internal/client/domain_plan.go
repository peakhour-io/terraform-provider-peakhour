package client

import "fmt"

// SetDomainPlanNew subscribes a domain to a new plan
func (c *Client) SetDomainPlanNew(domainName, planCode string) error {
	body := DomainPlanAdd{Code: planCode}
	return c.Put(fmt.Sprintf("/api/v1/domains/%s/plan/new", domainName), body)
}

// SetDomainPlanExisting subscribes a domain to an existing account plan
func (c *Client) SetDomainPlanExisting(domainName, planCode string) error {
	body := DomainPlanAdd{Code: planCode}
	return c.Put(fmt.Sprintf("/api/v1/domains/%s/plan", domainName), body)
}

// GetDomainPlan retrieves the current plan assigned to a domain
func (c *Client) GetDomainPlan(domainName string) (*DomainPlan, error) {
	var result DomainPlan
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/plan", domainName), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UnsubscribeDomainPlan removes the plan subscription from a domain
func (c *Client) UnsubscribeDomainPlan(domainName string) error {
	return c.Delete(fmt.Sprintf("/api/v1/domains/%s/plan", domainName), nil)
}
