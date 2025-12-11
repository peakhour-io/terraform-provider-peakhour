package provider

import (
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
