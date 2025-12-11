package provider

import "testing"

func TestImportIDParsing_RuleList(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "ok", input: "example.com/1234", wantErr: false},
		{name: "missing uuid", input: "example.com", wantErr: true},
		{name: "extra segment", input: "example.com/1234/extra", wantErr: true},
		{name: "empty domain", input: "/1234", wantErr: true},
		{name: "empty uuid", input: "example.com/", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseCompositeID(tc.input, 2)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseCompositeID(%q, 2) err=%v wantErr=%v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestImportIDParsing_RateLimitZone(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "ok", input: "example.com/my-zone", wantErr: false},
		{name: "missing name", input: "example.com", wantErr: true},
		{name: "extra segment", input: "example.com/my-zone/extra", wantErr: true},
		{name: "empty domain", input: "/my-zone", wantErr: true},
		{name: "empty name", input: "example.com/", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseCompositeID(tc.input, 2)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseCompositeID(%q, 2) err=%v wantErr=%v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestImportIDParsing_OriginPool(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		wantErr     bool
		wantOrigins bool
	}{
		{name: "ok", input: "example.com/origins/backend", wantErr: false, wantOrigins: true},
		{name: "wrong middle", input: "example.com/not-origins/backend", wantErr: false, wantOrigins: false},
		{name: "missing tag", input: "example.com/origins", wantErr: true},
		{name: "extra segment", input: "example.com/origins/backend/extra", wantErr: true},
		{name: "empty tag", input: "example.com/origins/", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parts, err := parseCompositeID(tc.input, 3)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseCompositeID(%q, 3) err=%v wantErr=%v", tc.input, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			gotOrigins := parts[1] == "origins"
			if gotOrigins != tc.wantOrigins {
				t.Fatalf("middle segment == origins: got=%v want=%v (parts=%v)", gotOrigins, tc.wantOrigins, parts)
			}
		})
	}
}

func TestImportIDParsing_Rule(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "ok", input: "example.com/requests/1234", wantErr: false},
		{name: "missing uuid", input: "example.com/requests", wantErr: true},
		{name: "too many", input: "example.com/requests/1234/extra", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseCompositeID(tc.input, 3)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseCompositeID(%q, 3) err=%v wantErr=%v", tc.input, err, tc.wantErr)
			}
		})
	}
}
