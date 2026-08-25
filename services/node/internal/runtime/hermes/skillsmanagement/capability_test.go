package skillsmanagement

import (
	"errors"
	"strings"
	"testing"
)

func TestHermes0205SkillMutationsAreUnavailable(t *testing.T) {
	tests := []struct {
		operation MutationOperation
		code      MutationErrorCode
		reasonHas string
	}{
		{operation: MutationInstall, code: MutationInstallUnavailable, reasonHas: "approved-source"},
		{operation: MutationUpdate, code: MutationUpdateScanBypassUnsafe, reasonHas: "force=True"},
		{operation: MutationRemove, code: MutationRemoveUnavailable, reasonHas: "authoritative read-back"},
		{operation: MutationConfigure, code: MutationConfigureUnavailable, reasonHas: "interactive terminal"},
	}
	for _, test := range tests {
		t.Run(string(test.operation), func(t *testing.T) {
			capability := MutationCapabilityFor(QualifiedHermesVersion, test.operation)
			if capability.Available() {
				t.Fatal("mutation capability must remain unavailable")
			}
			if capability.Operation() != test.operation || capability.ErrorCode() != test.code {
				t.Fatalf("capability = (%q, %q), want (%q, %q)", capability.Operation(), capability.ErrorCode(), test.operation, test.code)
			}
			if !strings.Contains(capability.Reason(), test.reasonHas) {
				t.Fatalf("Reason() = %q, want substring %q", capability.Reason(), test.reasonHas)
			}

			err := RequireMutation(QualifiedHermesVersion, test.operation)
			if !errors.Is(err, ErrMutationUnavailable) {
				t.Fatalf("RequireMutation() error = %v, want ErrMutationUnavailable", err)
			}
			var unavailable *MutationUnavailableError
			if !errors.As(err, &unavailable) {
				t.Fatalf("RequireMutation() error type = %T, want *MutationUnavailableError", err)
			}
			if unavailable.ErrorCode() != test.code || unavailable.Operation() != test.operation || unavailable.Reason() != capability.Reason() {
				t.Fatalf("unavailable error does not preserve capability truth: %#v", unavailable)
			}
		})
	}
}

func TestUnknownVersionAndOperationFailClosed(t *testing.T) {
	versionCapability := MutationCapabilityFor("0.20.6", MutationInstall)
	if versionCapability.Available() || versionCapability.ErrorCode() != MutationVersionUnqualified {
		t.Fatalf("unqualified version capability = %#v", versionCapability)
	}
	unknownCapability := MutationCapabilityFor(QualifiedHermesVersion, MutationOperation("force-update"))
	if unknownCapability.Available() || unknownCapability.ErrorCode() != MutationOperationUnknown {
		t.Fatalf("unknown operation capability = %#v", unknownCapability)
	}
}
