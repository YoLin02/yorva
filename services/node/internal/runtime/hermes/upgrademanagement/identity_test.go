package upgrademanagement

import (
	"errors"
	"testing"
)

func TestSnapshotIdentityRequiresExactCommitArchiveLicenseAndSource(t *testing.T) {
	identity := testIdentity("0.20.5", 'b')
	if err := identity.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*SnapshotIdentity)
	}{
		{name: "version", mutate: func(i *SnapshotIdentity) { i.Version = "0.20" }},
		{name: "commit", mutate: func(i *SnapshotIdentity) { i.Commit = "moving-main" }},
		{name: "archive size", mutate: func(i *SnapshotIdentity) { i.Archive.SizeBytes = 0 }},
		{name: "archive digest", mutate: func(i *SnapshotIdentity) { i.Archive.SHA256 = "ABC" }},
		{name: "license path", mutate: func(i *SnapshotIdentity) { i.LicensePath = "../LICENSE" }},
		{name: "license digest", mutate: func(i *SnapshotIdentity) { i.License.SHA256 = "" }},
		{name: "repository", mutate: func(i *SnapshotIdentity) { i.Source.Repository = "moving-url" }},
		{name: "archive root", mutate: func(i *SnapshotIdentity) { i.Source.ArchiveRoot = "../root" }},
		{name: "installer path", mutate: func(i *SnapshotIdentity) { i.Source.InstallerPath = `C:\\install.ps1` }},
		{name: "installer digest", mutate: func(i *SnapshotIdentity) { i.Source.Installer.SHA256 = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := identity
			test.mutate(&changed)
			if err := changed.Validate(); !errors.Is(err, ErrIdentityInvalid) {
				t.Fatalf("Validate() error = %v, want ErrIdentityInvalid", err)
			}
		})
	}
}

func TestZeroSnapshotIdentityIsUnknownAndExactEqualityUsesEveryField(t *testing.T) {
	if err := (SnapshotIdentity{}).Validate(); !errors.Is(err, ErrIdentityUnknown) {
		t.Fatalf("zero identity error = %v, want ErrIdentityUnknown", err)
	}
	left := testIdentity("0.20.5", 'b')
	right := left
	if !left.Equal(right) {
		t.Fatal("identical snapshots did not compare equal")
	}
	right.License.SizeBytes++
	if left.Equal(right) {
		t.Fatal("license mismatch was treated as the same snapshot")
	}
}
