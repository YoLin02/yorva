package install

// TransactionState is the sole persisted in-flight/recovery state.
// It is not an SQLite Operation status and not an activation pointer.
type TransactionState string

const (
	StateCreated    TransactionState = "CREATED"
	StateBuilding   TransactionState = "BUILDING"
	StateSealed     TransactionState = "SEALED"
	StatePublished  TransactionState = "PUBLISHED"
	StateActivating TransactionState = "ACTIVATING"
	StateCommitted  TransactionState = "COMMITTED"
	StateFailed     TransactionState = "FAILED"
)

func (s TransactionState) Terminal() bool {
	return s == StateCommitted || s == StateFailed
}

func (s TransactionState) Nonterminal() bool {
	return s != "" && !s.Terminal()
}

// Gate is the daemon install-subsystem admission state after recovery.
type Gate string

const (
	GateReady         Gate = "READY"
	GateReconciling   Gate = "RECONCILING"
	GateBlockedUnsafe Gate = "BLOCKED_UNSAFE"
)

// Action is a single recovery mutation. DecideRecovery never performs it.
type Action string

const (
	ActionNone                Action = "NONE"
	ActionFailTransaction     Action = "FAIL_TRANSACTION"
	ActionMoveStagingToFailed Action = "MOVE_STAGING_TO_FAILED"
	ActionRemoveEmptyStaging  Action = "REMOVE_EMPTY_STAGING"
	ActionPublish             Action = "PUBLISH"
	ActionPersistActivating   Action = "PERSIST_ACTIVATING"
	ActionActivate            Action = "ACTIVATE"
	ActionReconcileEnv        Action = "RECONCILE_ENV"
	ActionCommit              Action = "COMMIT"
	ActionGCStaging           Action = "GC_STAGING"
	ActionDiagnoseRetain      Action = "DIAGNOSE_RETAIN"
	ActionBlockUnsafe         Action = "BLOCK_UNSAFE"
	ActionFailFailableExtras  Action = "FAIL_FAILABLE_EXTRAS"
)

const (
	CodeInterrupted     = "OPERATION_INTERRUPTED"
	CodeBlockedUnsafe   = "INSTALL_BLOCKED_UNSAFE"
	CodeNotReady        = "INSTALL_NOT_READY"
	CodeInconsistent    = "INSTALL_STATE_INCONSISTENT"
	CodeSealInvalid     = "INSTALL_SEAL_INVALID"
	CodePublishConflict = "INSTALL_PUBLISH_CONFLICT"
)

// TransactionView is a classified transaction record. Recovery never reads SQLite Operations.
type TransactionView struct {
	Valid                bool
	OccupiesReservedName bool
	ID                   string
	State                TransactionState
	GenerationID         string
	ActiveBefore         string
	StagingRel           string
	GenerationRel        string
}

// TreeObservation is the observed directory at a transaction's expected path.
type TreeObservation struct {
	Present      bool
	Empty        bool
	Sealed       bool
	LineageMatch bool
	LineageID    string
}

// ActivePointer is the observed control/active.json. Missing/invalid is not "newest generation".
type ActivePointer struct {
	Present      bool
	Valid        bool
	GenerationID string
}

// EnvironmentObservation is derived-state input, never an activation source.
type EnvironmentObservation struct {
	HermesHomeOK bool
	PathOK       bool
}

func (e EnvironmentObservation) Complete() bool {
	return e.HermesHomeOK && e.PathOK
}

// Observation is the pure input to DecideRecovery.
type Observation struct {
	Transactions        []TransactionView
	Staging             TreeObservation
	Generation          TreeObservation
	UnknownDirectories  []string
	ReservedIDCollision bool
	Active              ActivePointer
	Environment         EnvironmentObservation
}

// RecoveryDecision is the pure output of DecideRecovery.
type RecoveryDecision struct {
	Gate      Gate
	NextState TransactionState
	Action    Action
	ErrorCode string
}
