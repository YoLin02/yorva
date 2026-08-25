package mcpmanagement

import (
	"errors"
	"reflect"
	"testing"
)

func TestReviewedRegistryStartsEmptyAndUnknownFailsClosed(t *testing.T) {
	registry := NewRegistry()
	if got := registry.Catalog(); len(got) != 0 {
		t.Fatalf("qualified catalog = %#v, want zero entries", got)
	}
	if _, err := registry.Resolve("unreviewed"); !errors.Is(err, ErrDescriptorUnknown) {
		t.Fatalf("Resolve(unreviewed) error = %v, want ErrDescriptorUnknown", err)
	}
	if _, err := registry.Resolve("https://caller.example/mcp"); !errors.Is(err, ErrDescriptorUnknown) {
		t.Fatalf("Resolve(caller URL) error = %v, want ErrDescriptorUnknown", err)
	}
}

func TestDescriptorSchemaAcceptsOnlyFixedHTTPSNoAuthOrBearer(t *testing.T) {
	noAuth := testDescriptor(CredentialClassNone)
	if err := validateDescriptor(noAuth); err != nil {
		t.Fatalf("validate no-auth descriptor: %v", err)
	}
	bearer := testDescriptor(CredentialClassStaticBearer)
	if err := validateDescriptor(bearer); err != nil {
		t.Fatalf("validate static-bearer descriptor: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*reviewedDescriptor)
	}{
		{name: "HTTP", mutate: func(d *reviewedDescriptor) { d.endpoint = "http://mcp.invalid.example/v1" }},
		{name: "userinfo", mutate: func(d *reviewedDescriptor) { d.endpoint = "https://token@mcp.invalid.example/v1" }},
		{name: "query", mutate: func(d *reviewedDescriptor) { d.endpoint = "https://mcp.invalid.example/v1?token=x" }},
		{name: "fragment", mutate: func(d *reviewedDescriptor) { d.endpoint = "https://mcp.invalid.example/v1#token" }},
		{name: "unclean path", mutate: func(d *reviewedDescriptor) { d.endpoint = "https://mcp.invalid.example/a/../v1" }},
		{name: "unknown auth", mutate: func(d *reviewedDescriptor) { d.credential = CredentialClass("OAUTH") }},
		{name: "no-auth key", mutate: func(d *reviewedDescriptor) { d.credential = CredentialClassNone; d.credentialKey = "MCP_TEST_API_KEY" }},
		{name: "bearer missing key", mutate: func(d *reviewedDescriptor) { d.credential = CredentialClassStaticBearer; d.credentialKey = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			descriptor := bearer
			test.mutate(&descriptor)
			if err := validateDescriptor(descriptor); !errors.Is(err, ErrDescriptorInvalid) {
				t.Fatalf("validateDescriptor() error = %v, want ErrDescriptorInvalid", err)
			}
		})
	}
}

func TestDescriptorHasNoGenericExecutionOrCallerOverrideFields(t *testing.T) {
	typeOfDescriptor := reflect.TypeOf(reviewedDescriptor{})
	forbidden := map[string]struct{}{
		"command": {}, "argv": {}, "args": {}, "env": {}, "environment": {},
		"header": {}, "headers": {}, "path": {}, "package": {}, "bootstrap": {},
		"redirect": {}, "oauth": {},
	}
	for index := 0; index < typeOfDescriptor.NumField(); index++ {
		field := typeOfDescriptor.Field(index)
		if _, found := forbidden[field.Name]; found {
			t.Fatalf("reviewed descriptor contains forbidden generic field %q", field.Name)
		}
	}
}

func testDescriptor(class CredentialClass) reviewedDescriptor {
	descriptor := reviewedDescriptor{
		presetID:     "unit-test-only",
		displayName:  "Unit test only",
		endpoint:     "https://mcp.invalid.example/v1",
		credential:   class,
		allowedTools: []string{"read", "search"},
	}
	if class == CredentialClassStaticBearer {
		descriptor.credentialKey = "MCP_UNIT_TEST_API_KEY"
	}
	return descriptor
}
