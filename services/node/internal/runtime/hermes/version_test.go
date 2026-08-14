package hermes

import "testing"

func TestParseVersionBanner(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		want      string
		supported bool
		wantError bool
	}{
		{name: "supported release", output: "Hermes Agent v0.19.0 (2026.7.20)\n", want: "0.19.0", supported: true},
		{name: "supported patch", output: "Hermes Agent 0.19.42\n", want: "0.19.42", supported: true},
		{name: "older release", output: "Hermes Agent v0.18.2 (2026.7.7.2)\n", want: "0.18.2"},
		{name: "newer minor", output: "Hermes Agent v0.20.0\n", want: "0.20.0"},
		{name: "prerelease", output: "Hermes Agent v0.19.1-rc.1\n", want: "0.19.1-rc.1"},
		{name: "documented date ignored", output: "Hermes Agent v0.19.3 (2026.8.14)\n", want: "0.19.3", supported: true},
		{name: "missing package banner", output: "2026.8.14\n", wantError: true},
		{name: "partial", output: "Hermes Agent v0.19\n", wantError: true},
		{name: "ambiguous", output: "Hermes Agent v0.19.0\nHermes Agent v0.19.1\n", wantError: true},
		{name: "leading zero", output: "Hermes Agent v0.019.0\n", wantError: true},
		{name: "overflow", output: "Hermes Agent v0.19.4294967296\n", wantError: true},
		{name: "invalid prerelease", output: "Hermes Agent v0.19.0-rc..1\n", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseVersionBanner(test.output)
			if test.wantError {
				if err == nil {
					t.Fatalf("parseVersionBanner(%q) succeeded: %#v", test.output, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseVersionBanner(%q) error = %v", test.output, err)
			}
			if got.String() != test.want || got.supported() != test.supported {
				t.Fatalf("parseVersionBanner(%q) = %q supported=%v, want %q supported=%v", test.output, got.String(), got.supported(), test.want, test.supported)
			}
		})
	}
}

func TestSupportedRangeDocumentsCompatibilityPolicy(t *testing.T) {
	if supportedRange != ">=0.19.0 <0.20.0" {
		t.Fatalf("supportedRange = %q", supportedRange)
	}
}
