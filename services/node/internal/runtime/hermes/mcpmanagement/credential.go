package mcpmanagement

import "errors"

const maxStaticBearerBytes = 4096

var (
	ErrCredentialClassInvalid  = errors.New("Hermes MCP credential class is invalid")
	ErrCredentialStatusInvalid = errors.New("Hermes MCP credential status is invalid")
	ErrCredentialInvalid       = errors.New("Hermes MCP credential is invalid")
)

type CredentialStatus string

const (
	CredentialStatusNotRequired   CredentialStatus = "NOT_REQUIRED"
	CredentialStatusNotConfigured CredentialStatus = "NOT_CONFIGURED"
	CredentialStatusConfigured    CredentialStatus = "CONFIGURED"
	CredentialStatusUnknown       CredentialStatus = "UNKNOWN"
)

// ValidateCredentialStatus verifies safe read-back metadata against the class
// owned by this reviewed descriptor.
func (s Selection) ValidateCredentialStatus(status CredentialStatus) error {
	if !s.valid() {
		return ErrDescriptorInvalid
	}
	return validateCredentialStatus(s.descriptor.credential, status)
}

// ValidateCredential validates a request-lifetime secret without retaining or
// returning it. The caller remains responsible for clearing its own buffer.
// No error includes any bytes from credential.
func (s Selection) ValidateCredential(credential []byte) error {
	if !s.valid() {
		return ErrDescriptorInvalid
	}
	switch s.descriptor.credential {
	case CredentialClassNone:
		if len(credential) != 0 {
			return ErrCredentialInvalid
		}
		return nil
	case CredentialClassStaticBearer:
		return validateStaticBearer(credential)
	default:
		return ErrCredentialClassInvalid
	}
}

func validateCredentialStatus(class CredentialClass, status CredentialStatus) error {
	switch class {
	case CredentialClassNone:
		if status != CredentialStatusNotRequired {
			return ErrCredentialStatusInvalid
		}
	case CredentialClassStaticBearer:
		if status != CredentialStatusNotConfigured && status != CredentialStatusConfigured && status != CredentialStatusUnknown {
			return ErrCredentialStatusInvalid
		}
	default:
		return ErrCredentialClassInvalid
	}
	return nil
}

func validateStaticBearer(credential []byte) error {
	if len(credential) == 0 || len(credential) > maxStaticBearerBytes {
		return ErrCredentialInvalid
	}
	for _, value := range credential {
		// A static Bearer value must be one printable ASCII header value. This
		// also excludes newline, NUL, whitespace and .env interpolation text.
		if value < 0x21 || value > 0x7e {
			return ErrCredentialInvalid
		}
	}
	return nil
}
