package skillsmanagement

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/managedskills"
)

func TestProjectInstallInspectListAndDefaultNamedIsolation(t *testing.T) {
	projector, home, managedRoot := newTestProjector(t)
	defaultSource := makeManagedSource(t, managedRoot, "instance-1", "demo", "default body")
	namedSource := makeManagedSource(t, managedRoot, "instance-1", "demo", "named body")

	defaultProjection, err := projector.Project(context.Background(), testProjectRequest("default", "demo", "1.0.0", defaultSource))
	if err != nil {
		t.Fatal(err)
	}
	if defaultProjection.DestinationDir != filepath.Join(home, "skills", "demo") {
		t.Fatalf("default destination = %q", defaultProjection.DestinationDir)
	}
	namedProjection, err := projector.Project(context.Background(), testProjectRequest("work", "demo", "1.0.0", namedSource))
	if err != nil {
		t.Fatal(err)
	}
	if namedProjection.DestinationDir != filepath.Join(home, "profiles", "work", "skills", "demo") {
		t.Fatalf("named destination = %q", namedProjection.DestinationDir)
	}
	if defaultProjection.ContentSHA256 == namedProjection.ContentSHA256 {
		t.Fatal("isolated profile fixtures unexpectedly have the same digest")
	}
	inspected, found, err := projector.Inspect(QualifiedHermesVersion, "default", "demo")
	if err != nil || !found || inspected != defaultProjection {
		t.Fatalf("inspect = %#v, %t, %v", inspected, found, err)
	}
	list, err := projector.List(QualifiedHermesVersion, "work")
	if err != nil || len(list) != 1 || list[0] != namedProjection {
		t.Fatalf("list = %#v, %v", list, err)
	}
}

func TestProjectRejectsExternalDestination(t *testing.T) {
	projector, home, managedRoot := newTestProjector(t)
	source := makeManagedSource(t, managedRoot, "instance-1", "demo", "body")
	external := filepath.Join(home, "skills", "demo")
	mustMkdirAll(t, external)
	mustWrite(t, filepath.Join(external, "SKILL.md"), "# external")

	_, err := projector.Project(context.Background(), testProjectRequest("default", "demo", "1.0.0", source))
	assertProjectionCode(t, err, ErrorDestinationConflict)
	if _, statErr := os.Stat(filepath.Join(external, "SKILL.md")); statErr != nil {
		t.Fatalf("external destination was removed: %v", statErr)
	}
}

func TestProjectUpdateReplacesVerifiedOwnedCopy(t *testing.T) {
	projector, _, managedRoot := newTestProjector(t)
	v1 := makeManagedSource(t, managedRoot, "instance-1", "demo", "version one")
	first, err := projector.Project(context.Background(), testProjectRequest("default", "demo", "1.0.0", v1))
	if err != nil {
		t.Fatal(err)
	}
	v2 := makeManagedSource(t, managedRoot, "instance-1", "demo", "version two")
	request := testProjectRequest("default", "demo", "2.0.0", v2)
	second, err := projector.Project(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.ContentSHA256 == second.ContentSHA256 || second.Version != "2.0.0" {
		t.Fatalf("update result = %#v after %#v", second, first)
	}
	content, err := os.ReadFile(filepath.Join(second.DestinationDir, "SKILL.md"))
	if err != nil || string(content) != skillBody("version two") {
		t.Fatalf("updated SKILL.md = %q, %v", content, err)
	}
}

func TestProjectAndUnprojectRefuseModifiedOwnedCopy(t *testing.T) {
	projector, _, managedRoot := newTestProjector(t)
	v1 := makeManagedSource(t, managedRoot, "instance-1", "demo", "version one")
	installed, err := projector.Project(context.Background(), testProjectRequest("default", "demo", "1.0.0", v1))
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(installed.DestinationDir, "SKILL.md"), skillBody("locally modified"))
	v2 := makeManagedSource(t, managedRoot, "instance-1", "demo", "version two")
	_, err = projector.Project(context.Background(), testProjectRequest("default", "demo", "2.0.0", v2))
	assertProjectionCode(t, err, ErrorProjectionModified)
	err = projector.Unproject(context.Background(), QualifiedHermesVersion, "default", "demo")
	assertProjectionCode(t, err, ErrorProjectionModified)
	content, readErr := os.ReadFile(filepath.Join(installed.DestinationDir, "SKILL.md"))
	if readErr != nil || string(content) != skillBody("locally modified") {
		t.Fatalf("modified projection was changed: %q, %v", content, readErr)
	}
}

func TestUnprojectRefusesExternalAndRemovesOwned(t *testing.T) {
	projector, home, managedRoot := newTestProjector(t)
	external := filepath.Join(home, "skills", "external")
	mustMkdirAll(t, external)
	mustWrite(t, filepath.Join(external, "SKILL.md"), "# external")
	err := projector.Unproject(context.Background(), QualifiedHermesVersion, "default", "external")
	assertProjectionCode(t, err, ErrorDestinationConflict)
	if _, statErr := os.Stat(external); statErr != nil {
		t.Fatalf("external destination was removed: %v", statErr)
	}

	source := makeManagedSource(t, managedRoot, "instance-1", "owned", "owned")
	projection, err := projector.Project(context.Background(), testProjectRequest("default", "owned", "1.0.0", source))
	if err != nil {
		t.Fatal(err)
	}
	if err := projector.Unproject(context.Background(), QualifiedHermesVersion, "default", "owned"); err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(projection.DestinationDir); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("owned destination still exists: %v", statErr)
	}
	if _, statErr := os.Stat(source); statErr != nil {
		t.Fatalf("SSOT source was removed by unproject: %v", statErr)
	}
}

func TestProjectRejectsCaseFoldCollisionAndTraversal(t *testing.T) {
	projector, home, managedRoot := newTestProjector(t)
	source := makeManagedSource(t, managedRoot, "instance-1", "demo", "body")
	mustMkdirAll(t, filepath.Join(home, "skills", "Demo"))
	_, err := projector.Project(context.Background(), testProjectRequest("default", "demo", "1.0.0", source))
	assertProjectionCode(t, err, ErrorDestinationConflict)

	request := testProjectRequest("default", "../demo", "1.0.0", source)
	_, err = projector.Project(context.Background(), request)
	assertProjectionCode(t, err, ErrorProjectionInvalid)
	outside := filepath.Join(filepath.Dir(home), "demo")
	if _, statErr := os.Stat(outside); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("traversal target exists: %v", statErr)
	}
}

func TestProjectRequiresExactVersionAndManagedDigestPath(t *testing.T) {
	projector, _, managedRoot := newTestProjector(t)
	source := makeManagedSource(t, managedRoot, "instance-1", "demo", "body")
	request := testProjectRequest("default", "demo", "1.0.0", source)
	request.RuntimeVersion = "0.20.6"
	_, err := projector.Project(context.Background(), request)
	assertProjectionCode(t, err, ErrorProjectionVersion)

	wrongPath := filepath.Join(managedRoot, "instance-1", "demo", "not-a-digest")
	mustMkdirAll(t, wrongPath)
	mustWrite(t, filepath.Join(wrongPath, "SKILL.md"), skillBody("body"))
	request = testProjectRequest("default", "demo", "1.0.0", wrongPath)
	_, err = projector.Project(context.Background(), request)
	assertProjectionCode(t, err, ErrorProjectionInvalid)
}

func newTestProjector(t *testing.T) (*Projector, string, string) {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "local", "hermes")
	managedRoot := filepath.Join(root, "data", "skills")
	mustMkdirAll(t, filepath.Join(home, "skills"))
	mustMkdirAll(t, filepath.Join(home, "profiles", "work"))
	mustMkdirAll(t, managedRoot)
	projector, err := NewProjector(home, managedRoot, "deployment-1")
	if err != nil {
		t.Fatal(err)
	}
	return projector, home, managedRoot
}

func makeManagedSource(t *testing.T, managedRoot, instanceID, skillID, body string) string {
	t.Helper()
	temporary, err := os.MkdirTemp(managedRoot, "staging-")
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(temporary, "SKILL.md"), skillBody(body))
	mustMkdirAll(t, filepath.Join(temporary, "references"))
	mustWrite(t, filepath.Join(temporary, "references", "usage.md"), "# Reference\n\nReviewed prose.\n")
	bundle, err := managedskills.InspectSourceDirectory(temporary)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(managedRoot, instanceID, skillID, bundle.ContentSHA256)
	mustMkdirAll(t, filepath.Dir(destination))
	if err := os.Rename(temporary, destination); err != nil {
		t.Fatal(err)
	}
	return destination
}

func testProjectRequest(profileID, skillID, version, source string) ProjectRequest {
	return ProjectRequest{RuntimeVersion: QualifiedHermesVersion, ProfileID: profileID, SkillID: skillID, SourceID: "reviewed", Version: version, SourceDir: source}
}

func skillBody(body string) string { return "# Demo\n\n" + body + "\n" }

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertProjectionCode(t *testing.T, err error, code ProjectionErrorCode) {
	t.Helper()
	var projectionErr *ProjectionError
	if !errors.As(err, &projectionErr) || projectionErr.Code != code {
		t.Fatalf("error = %v, want projection code %s", err, code)
	}
}
