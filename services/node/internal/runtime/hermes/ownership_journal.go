package hermes

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	promotionSchemaVersion = 1
	promoPrepared          = "PREPARED"
	promoOldQuarantined    = "OLD_QUARANTINED"
	promoNewPromoted       = "NEW_PROMOTED"
	promoCommitted         = "COMMITTED"
	promotionDirName       = ".yorva-phase3"
)

type promotionJournal struct {
	Schema            int    `json:"schema"`
	OperationID       string `json:"operationId"`
	SourcePin         string `json:"sourcePin"`
	Identity          string `json:"identity"`
	Target            string `json:"target"`
	Candidate         string `json:"candidate"`
	Quarantine        string `json:"quarantine"`
	PreviousManifest  string `json:"previousManifest"`
	CandidateManifest string `json:"candidateManifest"`
	State             string `json:"state"`
	CorrelationID     string `json:"correlationId"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
	MAC               string `json:"mac"`
}

func promotionJournalDir(home string) string {
	return filepath.Join(home, promotionDirName)
}

func promotionJournalPath(home, operationID string) string {
	safe := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, operationID)
	return filepath.Join(promotionJournalDir(home), "promote-"+safe+".json")
}

func journalMAC(nonce string, journal promotionJournal) string {
	mac := hmac.New(sha256.New, []byte(nonce))
	_, _ = mac.Write([]byte(strings.Join([]string{
		hex.EncodeToString([]byte{byte(journal.Schema)}),
		journal.OperationID,
		journal.SourcePin,
		journal.Identity,
		filepath.ToSlash(journal.Target),
		filepath.ToSlash(journal.Candidate),
		filepath.ToSlash(journal.Quarantine),
		journal.PreviousManifest,
		journal.CandidateManifest,
		journal.State,
		journal.CorrelationID,
	}, "\n")))
	return hex.EncodeToString(mac.Sum(nil))
}

func writePromotionJournal(ops atomicFileOps, home string, nonce string, journal promotionJournal) error {
	if err := os.MkdirAll(promotionJournalDir(home), 0o700); err != nil {
		return err
	}
	if err := rejectReparsePoint(promotionJournalDir(home)); err != nil {
		return err
	}
	journal.Schema = promotionSchemaVersion
	journal.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if journal.CreatedAt == "" {
		journal.CreatedAt = journal.UpdatedAt
	}
	journal.MAC = journalMAC(nonce, journal)
	payload, err := json.Marshal(journal)
	if err != nil {
		return err
	}
	return writeAtomicRegularFile(ops, promotionJournalPath(home, journal.OperationID), payload)
}

func readPromotionJournal(path string) (promotionJournal, error) {
	if err := rejectReparsePoint(path); err != nil {
		return promotionJournal{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return promotionJournal{}, err
	}
	if !info.Mode().IsRegular() || isReparsePoint(info) || info.Size() == 0 || info.Size() > maxOwnershipRecordBytes*4 {
		return promotionJournal{}, installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return promotionJournal{}, err
	}
	var journal promotionJournal
	if err := json.Unmarshal(payload, &journal); err != nil || journal.Schema != promotionSchemaVersion || journal.OperationID == "" {
		return promotionJournal{}, installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	return journal, nil
}

func verifyPromotionJournal(journal promotionJournal, identity ownershipIdentity) error {
	if journal.OperationID != identity.OperationID || journal.SourcePin != identity.SourcePin {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if identity.Target != "" && !sameCanonicalPath(journal.Target, identity.Target) {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if !hmac.Equal([]byte(strings.ToLower(journal.MAC)), []byte(journalMAC(identity.Nonce, journal))) {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	return nil
}

func listPromotionJournals(home string) ([]promotionJournal, error) {
	dir := promotionJournalDir(home)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var journals []promotionJournal
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "promote-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		journal, err := readPromotionJournal(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		journals = append(journals, journal)
	}
	return journals, nil
}
