//go:build windows

package skillsmanagement

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectRejectsWindowsReparseSource(t *testing.T) {
	projector, _, managedRoot := newTestProjector(t)
	realSource := makeManagedSource(t, managedRoot, "instance-1", "demo", "body")
	link := filepath.Join(managedRoot, "instance-1", "demo", "linked-source")
	if err := os.Symlink(realSource, link); err != nil {
		t.Skipf("creating a Windows directory symlink requires host permission: %v", err)
	}
	_, err := projector.Project(context.Background(), testProjectRequest("default", "demo", "1.0.0", link))
	assertProjectionCode(t, err, ErrorProjectionInvalid)
}
