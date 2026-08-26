//go:build mcpqualification

package hermes

import (
	"net/http"

	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/mcpmanagement"
)

// NewQualificationProfileMCPManager wires the fixed qualification-only preset
// to an isolated Hermes home. It exists only in mcpqualification-tagged test
// binaries and is never part of normal daemon or Desktop composition.
func NewQualificationProfileMCPManager(root string, client *http.Client) *ProfileMCPManager {
	return newProfileMCPManager(newProfileResourceReaderAt(root), mcpmanagement.NewQualificationRegistry(), client)
}
