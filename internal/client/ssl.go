package client

import "fmt"

func (c *Client) GetRPSSLConfig(domainName string) (*SSLConfig, error) {
	var result SSLConfig
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/ssl", domainName), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateRPSSLConfig(domainName string, config SSLConfig) error {
	return c.Put(fmt.Sprintf("/api/v1/domains/%s/services/rp/ssl", domainName), config)
}

func (c *Client) GetRPSSLCertificate(domainName string) (*SSLCertificate, error) {
	var result SSLCertificate
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/ssl/certificate", domainName), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateRPSSLCertificate(domainName string, cert SSLCertificateAdd) error {
	return c.Put(fmt.Sprintf("/api/v1/domains/%s/services/rp/ssl/certificate", domainName), cert)
}
