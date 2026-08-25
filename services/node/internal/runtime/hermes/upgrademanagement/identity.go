package upgrademanagement

import (
	"errors"
	"path"
	"regexp"
	"strings"
)

var (
	ErrIdentityInvalid = errors.New("Hermes snapshot identity is invalid")
	ErrIdentityUnknown = errors.New("Hermes snapshot identity is unknown")
)

var (
	versionPattern    = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	commitPattern     = regexp.MustCompile(`^[0-9a-f]{40}$`)
	sha256Pattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	rootPattern       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,191}$`)
	relativePattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_./-]{0,255}$`)
)

// ArtifactIdentity identifies immutable bytes. Size and SHA-256 are both
// required because neither one is a substitute for the other.
type ArtifactIdentity struct {
	SizeBytes int64
	SHA256    string
}

func (a ArtifactIdentity) validate() error {
	if a.SizeBytes <= 0 || !sha256Pattern.MatchString(a.SHA256) {
		return ErrIdentityInvalid
	}
	return nil
}

// SourceIdentity is the stable source provenance compiled into YORVA. It does
// not contain a transport URL: mirrors may change byte location but not this
// identity or the pinned installer inputs.
type SourceIdentity struct {
	Repository    string
	ArchiveRoot   string
	InstallerPath string
	Installer     ArtifactIdentity
}

func (s SourceIdentity) validate() error {
	if len(s.Repository) > 192 || !repositoryPattern.MatchString(s.Repository) ||
		!rootPattern.MatchString(s.ArchiveRoot) ||
		!safeRelativePath(s.InstallerPath) {
		return ErrIdentityInvalid
	}
	return s.Installer.validate()
}

// SnapshotIdentity is an exact Hermes source snapshot. Version text alone is
// deliberately insufficient: commit, archive, license, and installer source
// inputs all participate in equality.
type SnapshotIdentity struct {
	Version     string
	Commit      string
	Archive     ArtifactIdentity
	LicensePath string
	License     ArtifactIdentity
	Source      SourceIdentity
}

func (i SnapshotIdentity) Validate() error {
	if i == (SnapshotIdentity{}) {
		return ErrIdentityUnknown
	}
	if len(i.Version) > 32 || !versionPattern.MatchString(i.Version) ||
		!commitPattern.MatchString(i.Commit) ||
		!safeRelativePath(i.LicensePath) {
		return ErrIdentityInvalid
	}
	if err := i.Archive.validate(); err != nil {
		return err
	}
	if err := i.License.validate(); err != nil {
		return err
	}
	return i.Source.validate()
}

func (i SnapshotIdentity) Equal(other SnapshotIdentity) bool {
	return i == other
}

func safeRelativePath(value string) bool {
	if value == "" || strings.TrimSpace(value) != value || !relativePattern.MatchString(value) || strings.Contains(value, `\`) || strings.Contains(value, ":") {
		return false
	}
	cleaned := path.Clean(value)
	return cleaned == value && cleaned != "." && !strings.HasPrefix(cleaned, "/") && !strings.HasPrefix(cleaned, "../")
}
