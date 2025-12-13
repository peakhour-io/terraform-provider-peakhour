package client

import "fmt"

// GetRPConfig retrieves the reverse proxy raw configuration
func (c *Client) GetRPConfig(domainName string) (*RawConfig, error) {
	var result RawConfig
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/config", domainName), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListImageTransformPresets retrieves all image transform presets
func (c *Client) ListImageTransformPresets(domainName string) ([]ImageTransformPreset, error) {
	var result ImageTransformPresetList
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/image-transforms", domainName), &result)
	if err != nil {
		return nil, err
	}
	return result.Presets, nil
}

// GetImageTransformPreset retrieves a specific image transform preset
func (c *Client) GetImageTransformPreset(domainName, presetUUID string) (*ImageTransformPreset, error) {
	var result ImageTransformPreset
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/image-transforms/%s", domainName, presetUUID), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateImageTransformPreset creates a new image transform preset
func (c *Client) CreateImageTransformPreset(domainName string, preset ImageTransformPresetCreate) (*ImageTransformPreset, error) {
	var result ImageTransformPreset
	err := c.Post(fmt.Sprintf("/api/v1/domains/%s/image-transforms", domainName), preset, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateImageTransformPreset updates an existing image transform preset
func (c *Client) UpdateImageTransformPreset(domainName, presetUUID string, update ImageTransformPresetUpdate) (*ImageTransformPreset, error) {
	var result ImageTransformPreset
	err := c.Post(fmt.Sprintf("/api/v1/domains/%s/image-transforms/%s", domainName, presetUUID), update, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteImageTransformPreset deletes an image transform preset
func (c *Client) DeleteImageTransformPreset(domainName, presetUUID string) error {
	return c.Delete(fmt.Sprintf("/api/v1/domains/%s/image-transforms/%s", domainName, presetUUID), nil)
}

// CommitImageTransforms commits changes to image transform settings
func (c *Client) CommitImageTransforms(domainName string) error {
	return c.Post(fmt.Sprintf("/api/v1/domains/%s/image-transforms/config/commit", domainName), nil, nil)
}

// GetTransformSettings retrieves transform settings
func (c *Client) GetTransformSettings(domainName string) (*TransformSettings, error) {
	var result TransformSettings
	err := c.Get(fmt.Sprintf("/api/v1/domains/%s/services/rp/transforms", domainName), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateTransformSettings updates transform settings
func (c *Client) UpdateTransformSettings(domainName string, settings TransformSettings) error {
	return c.Patch(fmt.Sprintf("/api/v1/domains/%s/services/rp/transforms", domainName), settings)
}
