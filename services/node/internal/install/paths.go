package install

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	dirControl      = "control"
	dirTransactions = "transactions"
	dirGenerations  = "generations"
	dirStaging      = "staging"
	dirFailed       = "failed"
	fileActive      = "active.json"
	fileLock        = "install.lock"
	hermesHomeName  = "hermes"
)

// Layout is the managed filesystem under a Hermes home root.
// Production root is %LOCALAPPDATA%\hermes; tests inject a temp root.
type Layout struct {
	Root string
}

func DefaultManagedRoot() (string, error) {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return "", ErrManagedRootUnset
	}
	return filepath.Join(local, hermesHomeName), nil
}

func NewLayout(root string) (Layout, error) {
	if strings.TrimSpace(root) == "" {
		return Layout{}, ErrPathNotContained
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Layout{}, err
	}
	return Layout{Root: filepath.Clean(abs)}, nil
}

func StagingRel(txnID string) string {
	return dirStaging + "/" + txnID
}

func GenerationRel(genID string) string {
	return dirGenerations + "/" + genID
}

func (l Layout) ControlDir() string {
	return filepath.Join(l.Root, dirControl)
}

func (l Layout) TransactionsDir() string {
	return filepath.Join(l.ControlDir(), dirTransactions)
}

func (l Layout) ActivePath() string {
	return filepath.Join(l.ControlDir(), fileActive)
}

func (l Layout) LockPath() string {
	return filepath.Join(l.ControlDir(), fileLock)
}

func (l Layout) TransactionPath(id string) (string, error) {
	if err := ParseTransactionID(id); err != nil {
		return "", err
	}
	return l.ResolveContained(dirControl + "/" + dirTransactions + "/" + transactionFileName(id))
}

func (l Layout) StagingPath(txnID string) (string, error) {
	if err := ParseTransactionID(txnID); err != nil {
		return "", err
	}
	return l.ResolveContained(StagingRel(txnID))
}

func (l Layout) StagingRoot() string {
	return filepath.Join(l.Root, dirStaging)
}

func (l Layout) FailedRoot() string {
	return filepath.Join(l.Root, dirFailed)
}

func (l Layout) FailedPath(txnID string) (string, error) {
	if err := ParseTransactionID(txnID); err != nil {
		return "", err
	}
	return l.ResolveContained(dirFailed + "/" + txnID)
}

func (l Layout) GenerationsRoot() string {
	return filepath.Join(l.Root, dirGenerations)
}

func (l Layout) GenerationPath(genID string) (string, error) {
	if err := ParseGenerationID(genID); err != nil {
		return "", err
	}
	return l.ResolveContained(GenerationRel(genID))
}

func (l Layout) EnsureControl() error {
	if err := rejectReparse(l.Root); err != nil {
		return err
	}
	if err := os.MkdirAll(l.TransactionsDir(), 0o700); err != nil {
		return err
	}
	return rejectReparse(l.ControlDir())
}

func (l Layout) ResolveContained(rel string) (string, error) {
	if err := rejectUnsafeRel(rel); err != nil {
		return "", err
	}
	joined := filepath.Join(l.Root, filepath.FromSlash(rel))
	clean := filepath.Clean(joined)
	if !pathContained(l.Root, clean) {
		return "", ErrPathNotContained
	}
	return clean, nil
}

func ValidateStagingRel(rel, txnID string) error {
	if err := ParseTransactionID(txnID); err != nil {
		return err
	}
	return requireExactRel(rel, StagingRel(txnID))
}

func ValidateGenerationRel(rel, genID string) error {
	if err := ParseGenerationID(genID); err != nil {
		return err
	}
	return requireExactRel(rel, GenerationRel(genID))
}

func requireExactRel(rel, want string) error {
	if err := rejectUnsafeRel(rel); err != nil {
		return err
	}
	if rel != want {
		return ErrInvalidRelativePath
	}
	return nil
}

func rejectUnsafeRel(rel string) error {
	if rel == "" {
		return ErrInvalidRelativePath
	}
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, `\`) {
		return ErrPathNotContained
	}
	if len(rel) >= 2 && rel[1] == ':' {
		return ErrPathNotContained
	}
	if strings.ContainsAny(rel, `\`+"\x00") {
		return ErrInvalidRelativePath
	}
	for _, part := range strings.Split(rel, "/") {
		if part == "" || part == "." || part == ".." {
			return ErrInvalidRelativePath
		}
	}
	return nil
}

func pathContained(root, candidate string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func rejectReparse(path string) error {
	current := path
	for i := 0; i < 16; i++ {
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				parent := filepath.Dir(current)
				if parent == current {
					return nil
				}
				current = parent
				continue
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || isReparsePoint(info) {
			return ErrReparsePoint
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
	return nil
}
