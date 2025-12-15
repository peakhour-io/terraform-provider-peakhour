package provider

import (
	"encoding/json"
	"fmt"
	"strings"
)

func parseCompositeID(id string, parts int) ([]string, error) {
	segments := strings.Split(id, "/")
	if len(segments) != parts {
		return nil, fmt.Errorf("expected %d parts, got %d", parts, len(segments))
	}
	for _, s := range segments {
		if s == "" {
			return nil, fmt.Errorf("empty, expected %d parts separated by /", parts)
		}
	}
	return segments, nil
}

// isAlreadyExistsError checks if an API error indicates a resource already exists.
// This handles 400 errors with validation messages like "already exists".
func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "already exists")
}

func normalizeJSON(input string) (string, error) {
	var v interface{}
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		return "", err
	}
	output, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(output), nil
}
