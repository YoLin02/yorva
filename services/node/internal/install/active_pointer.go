package install

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"
)

const activeSchema = 1

// ActiveRecord is the on-disk control/active.json document.
type ActiveRecord struct {
	Schema                 int       `json:"schema"`
	RuntimeKind            string    `json:"runtimeKind"`
	GenerationID           string    `json:"generationId"`
	GenerationRelativePath string    `json:"generationRelativePath"`
	ManifestSHA256         string    `json:"manifestSha256"`
	SealSHA256             string    `json:"sealSha256"`
	SourcePin              string    `json:"sourcePin"`
	Version                string    `json:"version"`
	TransactionID          string    `json:"transactionId"`
	ActivatedAt            time.Time `json:"activatedAt"`
}

func (r ActiveRecord) Observation() ActivePointer {
	if err := validateActiveRecord(r); err != nil {
		return ActivePointer{Class: ActiveInvalid, Present: true}
	}
	return ActivePointer{
		Class:          ActiveValid,
		Present:        true,
		Valid:          true,
		GenerationID:   r.GenerationID,
		SealSHA256:     r.SealSHA256,
		ManifestSHA256: r.ManifestSHA256,
	}
}

func (s *Store) WriteActive(rec ActiveRecord) error {
	if err := validateActiveRecord(rec); err != nil {
		return err
	}
	if err := s.layout.EnsureControl(); err != nil {
		return err
	}
	if _, err := s.layout.ResolveContained(rec.GenerationRelativePath); err != nil {
		return err
	}
	payload, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return writeAtomicRecord(s.ops, s.layout.ActivePath(), payload)
}

func (s *Store) LoadActive() (ActiveRecord, error) {
	path := s.layout.ActivePath()
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ActiveRecord{}, ErrNotFound
		}
		return ActiveRecord{}, err
	}
	if !info.Mode().IsRegular() || isReparsePoint(info) || info.Size() > maxActiveBytes {
		return ActiveRecord{}, ErrInvalidRecord
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return ActiveRecord{}, err
	}
	var rec ActiveRecord
	if err := json.Unmarshal(payload, &rec); err != nil {
		return ActiveRecord{}, ErrInvalidRecord
	}
	if err := validateActiveRecord(rec); err != nil {
		return ActiveRecord{}, err
	}
	return rec, nil
}

func (s *Store) ReadActive() ActivePointer {
	rec, err := s.LoadActive()
	if errors.Is(err, ErrNotFound) {
		return ActivePointer{Class: ActiveMissing}
	}
	if err != nil {
		return ActivePointer{Class: ActiveInvalid, Present: true}
	}
	genAbs, err := s.layout.GenerationPath(rec.GenerationID)
	if err != nil {
		return ActivePointer{Class: ActiveInvalid, Present: true}
	}
	if err := VerifyPublishedGeneration(genAbs, rec.GenerationID, rec.ManifestSHA256, rec.SealSHA256); err != nil {
		return ActivePointer{Class: ActiveInvalid, Present: true}
	}
	got := rec.Observation()
	if !got.IsValid() {
		return ActivePointer{Class: ActiveInvalid, Present: true}
	}
	return got
}

func validateActiveRecord(r ActiveRecord) error {
	if r.Schema != activeSchema {
		return ErrInvalidRecord
	}
	if strings.TrimSpace(r.RuntimeKind) == "" || strings.TrimSpace(r.SourcePin) == "" || strings.TrimSpace(r.Version) == "" {
		return ErrInvalidRecord
	}
	if err := ParseGenerationID(r.GenerationID); err != nil {
		return err
	}
	if err := ParseTransactionID(r.TransactionID); err != nil {
		return err
	}
	if err := ValidateGenerationRel(r.GenerationRelativePath, r.GenerationID); err != nil {
		return err
	}
	if !validSHA256Hex(r.ManifestSHA256) || !validSHA256Hex(r.SealSHA256) {
		return ErrInvalidRecord
	}
	if r.ActivatedAt.IsZero() {
		return ErrInvalidRecord
	}
	return nil
}
