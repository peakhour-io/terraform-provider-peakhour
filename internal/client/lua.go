package client

import "fmt"

func (c *Client) GetRPLuaOptions(domainName string) (*LuaOptions, error) {
	var result LuaOptions
	if err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/lua", domainName), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateRPLuaOptions(domainName string, options LuaOptions) error {
	return c.Put(fmt.Sprintf("/api/v1/domains/%s/services/rp/lua", domainName), options)
}
