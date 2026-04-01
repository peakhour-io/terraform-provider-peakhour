package provider

import (
	"testing"
)

func TestParseCompositeID(t *testing.T) {
	tests := []struct {
		input    string
		parts    int
		expected []string
		wantErr  bool
	}{
		{"domain/phase/uuid", 3, []string{"domain", "phase", "uuid"}, false},
		{"domain/uuid", 2, []string{"domain", "uuid"}, false},
		{"domain/phase/uuid", 2, nil, true}, // expected 2, got 3
		{"domain//uuid", 3, nil, true},      // empty part
		{"", 3, nil, true},
	}

	for _, tt := range tests {
		got, err := parseCompositeID(tt.input, tt.parts)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseCompositeID(%q, %d) error = %v, wantErr %v", tt.input, tt.parts, err, tt.wantErr)
			continue
		}
		if !tt.wantErr {
			if len(got) != len(tt.expected) {
				t.Errorf("got len %d, expected %d", len(got), len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("got[%d] = %q, expected %q", i, got[i], tt.expected[i])
				}
			}
		}
	}
}

func TestNormalizeJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple normalization",
			input:    `{"a": 1, "b": 2}`,
			expected: `{"a":1,"b":2}`,
			wantErr:  false,
		},
		{
			name:     "whitespace and sorting",
			input:    `{  "b": 2, "a": 1 }`,
			expected: `{"a":1,"b":2}`,
			wantErr:  false,
		},
		{
			name:     "nested objects",
			input:    `{"outer": {"inner": "value"}}`,
			expected: `{"outer":{"inner":"value"}}`,
			wantErr:  false,
		},
		{
			name:     "complex rule action",
			input:    `{"firewall": [{"type": "firewall", "action": "deny"}]}`,
			expected: `{"firewall":[{"action":"deny","type":"firewall"}]}`,
			wantErr:  false,
		},
		{
			name:     "invalid json",
			input:    `{invalid`,
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizeJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("normalizeJSON() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNormalizeJSONDropNullObjectKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "drops null object keys",
			input:    `{"vconf":[{"type":"vconf","edge_ttl_sec":60,"auth_req":null,"blocklists":null}]}`,
			expected: `{"vconf":[{"edge_ttl_sec":60,"type":"vconf"}]}`,
			wantErr:  false,
		},
		{
			name:     "drops nested null keys",
			input:    `{"outer":{"inner":null,"inner2":{"x":null,"y":1}}}`,
			expected: `{"outer":{"inner2":{"y":1}}}`,
			wantErr:  false,
		},
		{
			name:     "preserves null array elements",
			input:    `{"arr":[null,{"a":null,"b":1}]}`,
			expected: `{"arr":[null,{"b":1}]}`,
			wantErr:  false,
		},
		{
			name:     "invalid json",
			input:    `{invalid`,
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeJSONDropNullObjectKeys(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizeJSONDropNullObjectKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("normalizeJSONDropNullObjectKeys() = %v, want %v", got, tt.expected)
			}
		})
	}
}
