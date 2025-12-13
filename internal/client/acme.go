package client

import "fmt"

func (c *Client) GetAcmeSettings(domainName string) (*AcmeSettings, error) {
	var result AcmeSettings
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/acme/settings", domainName), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateAcmeSettings(domainName string, settings AcmeSettings) error {
	return c.Put(fmt.Sprintf("/api/v1/domains/%s/services/acme/settings", domainName), settings)
}

func (c *Client) GetAcmeCertificate(domainName string) (*AcmeCertificate, error) {
	var result AcmeCertificate
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/acme/certificate", domainName), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// IssueAcmeCertificate triggers ACME certificate issuance/renewal for a domain.
func (c *Client) IssueAcmeCertificate(domainName string) error {
	return c.Post(fmt.Sprintf("/api/v1/domains/%s/services/acme/certificate", domainName), nil, nil)
}
