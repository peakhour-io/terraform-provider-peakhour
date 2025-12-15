package client

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type RedirectStatusCode int

func (c *RedirectStatusCode) UnmarshalJSON(b []byte) error {
	var n int
	if err := json.Unmarshal(b, &n); err == nil {
		*c = RedirectStatusCode(n)
		return nil
	}

	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("invalid redirect status_code %q: %w", s, err)
		}
		*c = RedirectStatusCode(n)
		return nil
	}

	return fmt.Errorf("invalid redirect status_code: %s", string(b))
}

func (c RedirectStatusCode) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(c))
}

// StringOrInt handles API fields that can be returned as either string or int
type StringOrInt string

func (s *StringOrInt) UnmarshalJSON(b []byte) error {
	// Try number first
	var n int
	if err := json.Unmarshal(b, &n); err == nil {
		*s = StringOrInt(strconv.Itoa(n))
		return nil
	}

	// Try string
	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		*s = StringOrInt(str)
		return nil
	}

	return fmt.Errorf("invalid value (expected string or int): %s", string(b))
}

func (s StringOrInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

func (s StringOrInt) String() string {
	return string(s)
}

type BulkRedirectListSummary struct {
	UUID         string  `json:"uuid"`
	Name         string  `json:"name"`
	Description  *string `json:"description"`
	EntriesCount int     `json:"entries_count"`
}

type BulkRedirectList struct {
	UUID         string  `json:"uuid"`
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	EntriesCount int     `json:"entries_count"`
}

type BulkRedirectListCreate struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type BulkRedirectListUpdate struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type BulkRedirectEntry struct {
	ID                  StringOrInt         `json:"id"`
	Enabled             *bool               `json:"enabled,omitempty"`
	PreserveQueryString *bool               `json:"preserve_query_string,omitempty"`
	SourceDomain        *string             `json:"source_domain,omitempty"`
	SourcePath          *string             `json:"source_path,omitempty"`
	SourceScheme        *string             `json:"source_scheme,omitempty"`
	StatusCode          *RedirectStatusCode `json:"status_code,omitempty"`
	TargetURL           *string             `json:"target_url,omitempty"`
}

type BulkRedirectEntryCreate struct {
	Enabled             *bool               `json:"enabled"`
	PreserveQueryString *bool               `json:"preserve_query_string"`
	SourceDomain        *string             `json:"source_domain"`
	SourcePath          *string             `json:"source_path"`
	SourceScheme        *string             `json:"source_scheme"`
	StatusCode          *RedirectStatusCode `json:"status_code"`
	TargetURL           *string             `json:"target_url"`
}

type BulkRedirectEntryUpdate struct {
	Enabled             *bool               `json:"enabled"`
	PreserveQueryString *bool               `json:"preserve_query_string"`
	SourceDomain        *string             `json:"source_domain"`
	SourcePath          *string             `json:"source_path"`
	SourceScheme        *string             `json:"source_scheme"`
	StatusCode          *RedirectStatusCode `json:"status_code"`
	TargetURL           *string             `json:"target_url"`
}

type BulkRedirectEntryBulkCreate struct {
	Entries []BulkRedirectEntryCreate `json:"entries"`
}
