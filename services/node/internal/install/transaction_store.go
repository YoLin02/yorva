package install

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
)

// Store persists transaction records and the active pointer under a managed root.
// It does not run Hermes builds, discovery, or GC.
type Store struct {
	layout Layout
	ops    atomicOps
}

func NewStore(root string) (*Store, error) {
	return newStoreWithOps(root, defaultAtomicOps())
}

func newStoreWithOps(root string, ops atomicOps) (*Store, error) {
	layout, err := NewLayout(root)
	if err != nil {
		return nil, err
	}
	return &Store{layout: layout, ops: ops}, nil
}

func (s *Store) Layout() Layout {
	return s.layout
}

func (s *Store) SaveTransaction(txn InstallTransaction) error {
	if err := validateTransaction(txn); err != nil {
		return err
	}
	if err := s.layout.EnsureControl(); err != nil {
		return err
	}
	dest, err := s.layout.TransactionPath(txn.ID)
	if err != nil {
		return err
	}
	current, err := s.LoadTransaction(txn.ID)
	found := err == nil
	if err != nil && !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrInvalidRecord) {
		return err
	}
	if errors.Is(err, ErrInvalidRecord) {
		return ErrInvalidRecord
	}
	var expected uint64
	if found {
		expected = current.Revision
	}
	if txn.Revision != expected {
		return ErrRevisionConflict
	}
	toWrite := txn
	toWrite.Revision = expected + 1
	payload, err := json.Marshal(toWrite)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return writeAtomicRecord(s.ops, dest, payload)
}

func (s *Store) LoadTransaction(id string) (InstallTransaction, error) {
	dest, err := s.layout.TransactionPath(id)
	if err != nil {
		return InstallTransaction{}, err
	}
	payload, err := os.ReadFile(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return InstallTransaction{}, ErrNotFound
		}
		return InstallTransaction{}, err
	}
	return decodeTransaction(payload)
}

func (s *Store) ListTransactionViews() ([]TransactionView, bool, error) {
	if err := s.layout.EnsureControl(); err != nil {
		return nil, false, err
	}
	return s.inspectTransactions()
}

func (s *Store) inspectTransactions() ([]TransactionView, bool, error) {
	entries, err := os.ReadDir(s.layout.TransactionsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	seen := make(map[string]struct{})
	collision := false
	var views []TransactionView
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".yorva-atom-") {
			continue
		}
		if !OccupiesReservedTxnName(name) {
			continue
		}
		view := TransactionView{OccupiesReservedName: true, ID: strings.TrimSuffix(name, txnFileSuffix)}
		id := strings.TrimSuffix(name, txnFileSuffix)
		if ParseTransactionID(id) != nil || !strings.HasSuffix(name, txnFileSuffix) || entry.IsDir() {
			views = append(views, view)
			continue
		}
		txn, err := s.LoadTransaction(id)
		if err != nil {
			views = append(views, view)
			continue
		}
		if txn.ID != id {
			views = append(views, view)
			continue
		}
		if _, dup := seen[id]; dup {
			collision = true
			views = append(views, view)
			continue
		}
		seen[id] = struct{}{}
		views = append(views, txn.View())
	}
	return views, collision, nil
}
