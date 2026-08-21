package install

import (
	"context"
	"errors"
	"os"
	"time"
)

const (
	FailBeforeGenerationMkdir     = "before-generation-mkdir"
	FailAfterGenerationMkdir      = "after-generation-mkdir"
	FailAfterBuild                = "after-build"
	FailBeforeValidate            = "before-validate"
	FailDuringManifestWalk        = "during-manifest-walk"
	FailAfterManifest             = "after-manifest"
	FailBeforeGenerationJSON      = "before-generation-json"
	FailAfterSealBeforeSecondWalk = "after-seal-before-second-walk"
	FailBeforePublishVerify       = "before-publish-verify"
	FailAfterPublishVerify        = "after-publish-verify"
	FailAfterPublished            = "after-published"
	FailBeforeActivatingPersist   = "before-activating-persist"
	FailAfterActivatingPersist    = "after-activating-persist"
	FailDuringActiveWrite         = "during-active-write"
	FailAfterActiveWrite          = "after-active-write"
)

// Legacy aliases keep older failpoint callers source-compatible while the
// implementation no longer builds in staging or renames at publish.
const (
	FailBeforeStagingMkdir  = FailBeforeGenerationMkdir
	FailAfterStagingMkdir   = FailAfterGenerationMkdir
	FailBeforePublishRename = FailBeforePublishVerify
	FailAfterPublishRename  = FailAfterPublishVerify
)

type CandidateBuildFunc func(ctx context.Context, generationAbs, hermesHome string) error

type CandidateValidateFunc func(ctx context.Context, generationAbs, expectedVersion string) error

// Manager runs CREATED → BUILDING → SEALED → PUBLISHED → ACTIVATING → COMMITTED.
type Manager struct {
	store    *Store
	build    CandidateBuildFunc
	validate CandidateValidateFunc
	now      func() time.Time
	failAt   string
	env      EnvironmentStore
}

func NewManager(store *Store, build CandidateBuildFunc, validate CandidateValidateFunc) *Manager {
	return &Manager{
		store:    store,
		build:    build,
		validate: validate,
		now:      func() time.Time { return time.Now().UTC() },
		env:      defaultEnvironmentStore(),
	}
}

func (m *Manager) withFailpoint(name string) *Manager {
	clone := *m
	clone.failAt = name
	return &clone
}

func (m *Manager) withEnv(env EnvironmentStore) *Manager {
	return WithEnvironment(m, env)
}

func WithEnvironment(m *Manager, env EnvironmentStore) *Manager {
	clone := *m
	clone.env = env
	return &clone
}

func (m *Manager) failpoint(name string) error {
	if m.failAt == name {
		return errors.New("injected failpoint " + name)
	}
	return nil
}

// SealNew persists CREATED (unless already on disk), builds directly in a fresh
// inactive generations/gen_* directory, validates from that final path, and seals.
func (m *Manager) SealNew(ctx context.Context, txn InstallTransaction) (InstallTransaction, error) {
	if txn.State != StateCreated {
		return txn, ErrInvalidRecord
	}
	if txn.Revision == 0 {
		if err := m.persist(&txn); err != nil {
			return txn, err
		}
	} else {
		loaded, err := m.store.LoadTransaction(txn.ID)
		if err != nil {
			return txn, err
		}
		if loaded.State != StateCreated {
			return loaded, ErrInvalidRecord
		}
		txn = loaded
	}
	txn.State = StateBuilding
	txn.Step = "build"
	txn.UpdatedAt = m.now()
	if err := m.persist(&txn); err != nil {
		return txn, err
	}
	if err := m.failpoint(FailBeforeGenerationMkdir); err != nil {
		return m.fail(txn, CodeInterrupted)
	}
	generationAbs, err := m.createFreshGenerationCandidate(txn)
	if err != nil {
		return m.fail(txn, CodeInterrupted)
	}
	if err := m.failpoint(FailAfterGenerationMkdir); err != nil {
		return m.fail(txn, CodeInterrupted)
	}
	if err := ctx.Err(); err != nil {
		return m.fail(txn, CodeInterrupted)
	}
	if m.build != nil {
		if err := m.build(ctx, generationAbs, m.store.layout.Root); err != nil {
			return m.fail(txn, interruptCode(err))
		}
	}
	if err := m.failpoint(FailAfterBuild); err != nil {
		return m.fail(txn, CodeInterrupted)
	}
	if err := m.failpoint(FailBeforeValidate); err != nil {
		return m.fail(txn, CodeSealInvalid)
	}
	if m.validate != nil {
		if err := m.validate(ctx, generationAbs, txn.ExpectedVersion); err != nil {
			return m.fail(txn, CodeSealInvalid)
		}
	}
	candidate, ok := readCandidateRecord(generationAbs)
	if !ok || candidate.TransactionID != txn.ID || candidate.GenerationID != txn.GenerationID || candidate.RuntimeKind != txn.RuntimeKind {
		return m.fail(txn, CodeSealInvalid)
	}
	sealed, err := SealGeneration(m.store.ops, SealInput{
		RootAbs:                generationAbs,
		TransactionID:          txn.ID,
		GenerationID:           txn.GenerationID,
		RuntimeKind:            txn.RuntimeKind,
		SourcePin:              txn.SourcePin,
		ExpectedVersion:        txn.ExpectedVersion,
		GenerationRelativePath: txn.GenerationRelativePath,
		CreatedAt:              m.now(),
	}, m.sealHooks())
	if err != nil {
		return m.fail(txn, CodeSealInvalid)
	}
	now := m.now()
	txn.State = StateSealed
	txn.Step = "sealed"
	txn.ManifestSHA256 = sealed.ManifestSHA256
	txn.SealSHA256 = sealed.SealSHA256
	txn.SealedAt = &now
	txn.UpdatedAt = now
	txn.ErrorCode = ""
	if err := m.persist(&txn); err != nil {
		return txn, err
	}
	return txn, nil
}

// PublishAndActivate verifies the sealed final-path generation and writes active.json.
// Environment reconcile is Batch 5; this leaves the transaction ACTIVATING.
func (m *Manager) PublishAndActivate(ctx context.Context, txn InstallTransaction) (InstallTransaction, error) {
	if err := ctx.Err(); err != nil {
		return txn, err
	}
	loaded, err := m.store.LoadTransaction(txn.ID)
	if err != nil {
		return txn, err
	}
	switch loaded.State {
	case StateSealed, StatePublished, StateActivating:
	default:
		return loaded, ErrInvalidRecord
	}
	if err := m.publish(&loaded); err != nil {
		return m.publishError(loaded, err)
	}
	if err := m.activate(&loaded); err != nil {
		return m.activateError(loaded, err)
	}
	return loaded, nil
}

func (m *Manager) publishError(txn InstallTransaction, err error) (InstallTransaction, error) {
	if errors.Is(err, ErrBlockedUnsafe) {
		return txn, err
	}
	if errors.Is(err, ErrPublishConflict) || errors.Is(err, ErrSealInvalid) {
		return m.fail(txn, CodePublishConflict)
	}
	if m.failAt != "" {
		return txn, err
	}
	return txn, err
}

func (m *Manager) activateError(txn InstallTransaction, err error) (InstallTransaction, error) {
	if errors.Is(err, ErrBlockedUnsafe) {
		return txn, err
	}
	if m.failAt != "" {
		return txn, err
	}
	if errors.Is(err, ErrSealInvalid) {
		return txn, err
	}
	return txn, err
}

func (m *Manager) createFreshGenerationCandidate(txn InstallTransaction) (string, error) {
	if err := os.MkdirAll(m.store.layout.GenerationsRoot(), 0o700); err != nil {
		return "", err
	}
	if err := rejectReparse(m.store.layout.GenerationsRoot()); err != nil {
		return "", err
	}
	generationAbs, err := m.store.layout.GenerationPath(txn.GenerationID)
	if err != nil {
		return "", err
	}
	if _, err := os.Lstat(generationAbs); err == nil {
		return "", ErrGenerationOccupied
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.Mkdir(generationAbs, 0o700); err != nil {
		return "", err
	}
	if err := rejectReparse(generationAbs); err != nil {
		_ = os.Remove(generationAbs)
		return "", err
	}
	if err := writeCandidateRecord(m.store.ops, generationAbs, txn); err != nil {
		_ = os.RemoveAll(generationAbs)
		return "", err
	}
	return generationAbs, nil
}

func (m *Manager) persist(txn *InstallTransaction) error {
	expected := txn.Revision
	if err := m.store.SaveTransaction(*txn); err != nil {
		loaded, loadErr := m.store.LoadTransaction(txn.ID)
		if loadErr == nil && loaded.ID == txn.ID && loaded.Revision > expected {
			*txn = loaded
			return nil
		}
		return err
	}
	loaded, err := m.store.LoadTransaction(txn.ID)
	if err != nil {
		return err
	}
	*txn = loaded
	return nil
}

func (m *Manager) fail(txn InstallTransaction, code string) (InstallTransaction, error) {
	if rec, err := m.store.LoadActive(); err == nil && rec.GenerationID == txn.GenerationID {
		txn.State = StateActivating
		txn.ErrorCode = ""
		txn.UpdatedAt = m.now()
		if err := m.persist(&txn); err != nil {
			return txn, err
		}
		return txn, errors.New(code)
	}
	txn.State = StateFailed
	txn.ErrorCode = code
	txn.UpdatedAt = m.now()
	if err := m.persist(&txn); err != nil {
		return txn, err
	}
	return txn, errors.New(code)
}

func (m *Manager) sealHooks() sealHooks {
	return sealHooks{
		AfterWalk:        func() error { return m.failpoint(FailDuringManifestWalk) },
		AfterManifest:    func() error { return m.failpoint(FailAfterManifest) },
		BeforeGeneration: func() error { return m.failpoint(FailBeforeGenerationJSON) },
		BeforeSecondWalk: func() error { return m.failpoint(FailAfterSealBeforeSecondWalk) },
	}
}

func interruptCode(err error) string {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return CodeInterrupted
	}
	return CodeInterrupted
}
