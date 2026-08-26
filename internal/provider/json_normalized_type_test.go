package provider

import (
	"context"
	"testing"
)

func TestJSONNormalizedValueStringSemanticEquals(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		proposed string
		prior    string
		want     bool
	}{
		"server defaults are ignored": {
			proposed: `{"methods":{"allowed_methods":["GET","POST"]},"protocol":{"max_num_args":100,"allowed_http_versions":["HTTP/1.1","HTTP/2"]},"initialization":{"blocking_paranoia_level":1}}`,
			prior:    `{"methods":{"allowed_methods":["GET","POST"]},"protocol":{"max_num_args":100}}`,
			want:     true,
		},
		"configured scalar drift is detected": {
			proposed: `{"protocol":{"max_num_args":200,"allowed_http_versions":["HTTP/1.1"]}}`,
			prior:    `{"protocol":{"max_num_args":100}}`,
			want:     false,
		},
		"missing configured field is detected": {
			proposed: `{"protocol":{}}`,
			prior:    `{"protocol":{"max_num_args":100}}`,
			want:     false,
		},
		"arrays must match exactly": {
			proposed: `{"methods":{"allowed_methods":["GET","POST","OPTIONS"]}}`,
			prior:    `{"methods":{"allowed_methods":["GET","POST"]}}`,
			want:     false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			proposed := NewJSONNormalizedValue(test.proposed)
			prior := NewJSONNormalizedValue(test.prior)

			got, diags := proposed.StringSemanticEquals(context.Background(), prior)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if got != test.want {
				t.Fatalf("StringSemanticEquals() = %t, want %t", got, test.want)
			}
		})
	}
}
