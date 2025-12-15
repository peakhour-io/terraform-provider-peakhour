package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultAPIURL = "https://www.peakhour.io"

type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
	Headers    map[string]string
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	type validationError struct {
		Loc  []any  `json:"loc"`
		Msg  string `json:"msg"`
		Type string `json:"type"`
	}

	type errorPayload struct {
		Error            string            `json:"error"`
		ValidationErrors []validationError `json:"validation_errors"`
		Detail           any               `json:"detail"`
	}

	var payload errorPayload
	if err := json.Unmarshal([]byte(e.Body), &payload); err == nil {
		if payload.Error != "" {
			if len(payload.ValidationErrors) == 0 {
				return fmt.Sprintf("API error (status %d): %s", e.StatusCode, payload.Error)
			}

			lines := make([]string, 0, len(payload.ValidationErrors))
			for _, ve := range payload.ValidationErrors {
				loc := formatValidationLoc(ve.Loc)
				msg := strings.TrimSpace(ve.Msg)

				switch {
				case loc == "" && msg != "":
					lines = append(lines, msg)
				case msg == "":
					lines = append(lines, loc)
				case ve.Type != "":
					lines = append(lines, fmt.Sprintf("%s: %s (%s)", loc, msg, ve.Type))
				default:
					lines = append(lines, fmt.Sprintf("%s: %s", loc, msg))
				}
			}

			return fmt.Sprintf("API error (status %d): %s\nValidation errors:\n- %s", e.StatusCode, payload.Error, strings.Join(lines, "\n- "))
		}

		if payload.Detail != nil {
			switch v := payload.Detail.(type) {
			case string:
				if strings.TrimSpace(v) != "" {
					return fmt.Sprintf("API error (status %d): %s", e.StatusCode, v)
				}
			}
		}
	}

	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Body)
}

func formatValidationLoc(loc []any) string {
	if len(loc) == 0 {
		return ""
	}

	parts := make([]string, 0, len(loc))
	for _, part := range loc {
		switch v := part.(type) {
		case string:
			parts = append(parts, v)
		case float64:
			if v == float64(int(v)) {
				parts = append(parts, fmt.Sprintf("%d", int(v)))
			} else {
				parts = append(parts, fmt.Sprintf("%v", v))
			}
		default:
			parts = append(parts, fmt.Sprint(v))
		}
	}

	return strings.Join(parts, ".")
}

func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

func IsNotFoundError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.IsNotFound()
	}
	return false
}

func (e *APIError) IsConflict() bool {
	return e.StatusCode == http.StatusConflict
}

func IsConflictError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.IsConflict()
	}
	return false
}

func NewClient(apiKey string, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultAPIURL
	}

	return &Client{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	for k, v := range c.Headers {
		if v == "" {
			continue
		}
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(bodyBytes)}
	}

	return resp, nil
}

func (c *Client) Get(path string, result interface{}) error {
	return c.GetWithContext(context.Background(), path, result)
}

func (c *Client) GetWithContext(ctx context.Context, path string, result interface{}) error {
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) Post(path string, body interface{}, result interface{}) error {
	return c.PostWithContext(context.Background(), path, body, result)
}

func (c *Client) PostWithContext(ctx context.Context, path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest(ctx, "POST", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) Patch(path string, body interface{}) error {
	return c.PatchWithContext(context.Background(), path, body)
}

func (c *Client) PatchWithContext(ctx context.Context, path string, body interface{}) error {
	resp, err := c.doRequest(ctx, "PATCH", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) Put(path string, body interface{}) error {
	return c.PutWithContext(context.Background(), path, body)
}

func (c *Client) PutWithContext(ctx context.Context, path string, body interface{}) error {
	resp, err := c.doRequest(ctx, "PUT", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) PutAndDecode(path string, body interface{}, result interface{}) error {
	return c.PutAndDecodeWithContext(context.Background(), path, body, result)
}

func (c *Client) PutAndDecodeWithContext(ctx context.Context, path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest(ctx, "PUT", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) Delete(path string, body interface{}) error {
	return c.DeleteWithContext(context.Background(), path, body)
}

func (c *Client) DeleteWithContext(ctx context.Context, path string, body interface{}) error {
	resp, err := c.doRequest(ctx, "DELETE", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
