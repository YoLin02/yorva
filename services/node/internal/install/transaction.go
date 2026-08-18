package install

import (
	"encoding/json"
	"strings"
	"time"
)

const transactionSchema = 1

// InstallTransaction is the sole persisted in-flight/recovery record.
type InstallTransaction struct {
	Schema                 int              `json:"schema"`
	Revision               uint64           `json:"revision"`
	ID                     string           `json:"id"`
	RuntimeKind            string           `json:"runtimeKind"`
	OperationID            string           `json:"operationId"`
	GenerationID           string           `json:"generationId"`
	State                  TransactionState `json:"state"`
	Step                   string           `json:"step"`
	SourcePin              string           `json:"sourcePin"`
	ExpectedVersion        string           `json:"expectedVersion"`
	StagingRelativePath    string           `json:"stagingRelativePath"`
	GenerationRelativePath string           `json:"generationRelativePath"`
	ManifestSHA256         string           `json:"manifestSha256"`
	SealSHA256             string           `json:"sealSha256"`
	ActiveBeforeGeneration string           `json:"activeBeforeGeneration"`
	ActiveBeforeDigest     string           `json:"activeBeforeDigest"`
	ErrorCode              string           `json:"errorCode"`
	CreatedAt              time.Time        `json:"createdAt"`
	UpdatedAt              time.Time        `json:"updatedAt"`
	SealedAt               *time.Time       `json:"sealedAt"`
	PublishedAt            *time.Time       `json:"publishedAt"`
	ActivatedAt            *time.Time       `json:"activatedAt"`
	CommittedAt            *time.Time       `json:"committedAt"`
}

func NewCreatedTransaction(runtimeKind, operationID, sourcePin, expectedVersion string) (InstallTransaction, error) {
	txnID, err := NewTransactionID()
	if err != nil {
		return InstallTransaction{}, err
	}
	genID, err := NewGenerationID()
	if err != nil {
		return InstallTransaction{}, err
	}
	now := time.Now().UTC()
	txn := InstallTransaction{
		Schema:                 transactionSchema,
		ID:                     txnID,
		RuntimeKind:            runtimeKind,
		OperationID:            operationID,
		GenerationID:           genID,
		State:                  StateCreated,
		SourcePin:              sourcePin,
		ExpectedVersion:        expectedVersion,
		StagingRelativePath:    StagingRel(txnID),
		GenerationRelativePath: GenerationRel(genID),
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if err := validateTransaction(txn); err != nil {
		return InstallTransaction{}, err
	}
	return txn, nil
}

func (t InstallTransaction) View() TransactionView {
	return TransactionView{
		Valid:         true,
		ID:            t.ID,
		State:         t.State,
		GenerationID:  t.GenerationID,
		ActiveBefore:  t.ActiveBeforeGeneration,
		StagingRel:    t.StagingRelativePath,
		GenerationRel: t.GenerationRelativePath,
	}
}

func validateTransaction(t InstallTransaction) error {
	if t.Schema != transactionSchema {
		return ErrInvalidRecord
	}
	if err := ParseTransactionID(t.ID); err != nil {
		return err
	}
	if err := ParseGenerationID(t.GenerationID); err != nil {
		return err
	}
	if !validState(t.State) {
		return ErrInvalidRecord
	}
	if strings.TrimSpace(t.RuntimeKind) == "" {
		return ErrInvalidRecord
	}
	if err := ValidateStagingRel(t.StagingRelativePath, t.ID); err != nil {
		return err
	}
	if err := ValidateGenerationRel(t.GenerationRelativePath, t.GenerationID); err != nil {
		return err
	}
	if t.ManifestSHA256 != "" && !validSHA256Hex(t.ManifestSHA256) {
		return ErrInvalidRecord
	}
	if t.SealSHA256 != "" && !validSHA256Hex(t.SealSHA256) {
		return ErrInvalidRecord
	}
	if t.ActiveBeforeGeneration != "" {
		if err := ParseGenerationID(t.ActiveBeforeGeneration); err != nil {
			return err
		}
	}
	if t.ActiveBeforeDigest != "" && !validSHA256Hex(t.ActiveBeforeDigest) {
		return ErrInvalidRecord
	}
	if t.CreatedAt.IsZero() || t.UpdatedAt.IsZero() {
		return ErrInvalidRecord
	}
	return nil
}

func validState(s TransactionState) bool {
	switch s {
	case StateCreated, StateBuilding, StateSealed, StatePublished, StateActivating, StateCommitted, StateFailed:
		return true
	default:
		return false
	}
}

func validSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func decodeTransaction(payload []byte) (InstallTransaction, error) {
	var txn InstallTransaction
	if err := json.Unmarshal(payload, &txn); err != nil {
		return InstallTransaction{}, ErrInvalidRecord
	}
	if err := validateTransaction(txn); err != nil {
		return InstallTransaction{}, err
	}
	return txn, nil
}
