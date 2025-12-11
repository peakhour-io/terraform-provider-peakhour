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
