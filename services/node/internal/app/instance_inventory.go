package app

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const hermesRuntimeID = "hermes"

var (
	ErrRuntimeNotSupported        = errors.New("runtime is not supported for instance inventory")
	ErrInstanceNotFound           = errors.New("instance not found")
	ErrInstanceQueryFailed        = errors.New("instance query failed")
	ErrInstanceOutputUnrecognized = errors.New("instance output unrecognized")
	ErrInstanceRuntimeNotFound    = errors.New("runtime inventory target not found")
)

type ProfileSnapshot struct {
	NativeID string
	Default  bool
}

type ProfileSource interface {
	List(ctx context.Context, executable string) ([]ProfileSnapshot, error)
}

type InstanceCapabilities struct {
	Instances bool `json:"instances"`
	Lifecycle bool `json:"lifecycle"`
}

type InstanceView struct {
	InstanceID            string
	RuntimeInstallationID string
	Name                  string
	Default               bool
	Protected             bool
	Availability          instance.Availability
	LastSyncedAt          *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Capabilities          InstanceCapabilities
}

type InstanceList struct {
	RuntimeID             string
	RuntimeInstallationID string
	Freshness             string
	LastSyncedAt          *time.Time
	Instances             []InstanceView
	Capabilities          InstanceCapabilities
	ErrorCode             yorvaruntime.ErrorCode
}

type InstanceInventory struct {
	discovery *RuntimeDiscovery
	db        *sqlite.Database
	source    ProfileSource
	nodeID    string
	now       func() time.Time
	mu        sync.Mutex
	locks     map[string]*sync.Mutex
}

func NewInstanceInventory(discovery *RuntimeDiscovery, db *sqlite.Database, source ProfileSource, nodeID string) *InstanceInventory {
	return &InstanceInventory{
		discovery: discovery,
		db:        db,
		source:    source,
		nodeID:    nodeID,
		now:       func() time.Time { return time.Now().UTC() },
		locks:     make(map[string]*sync.Mutex),
	}
}

func (s *InstanceInventory) ListInstances(ctx context.Context, runtimeID string) (InstanceList, error) {
	if runtimeID != hermesRuntimeID {
		return InstanceList{}, ErrInstanceRuntimeNotFound
	}
	if s.discovery == nil || s.db == nil || s.source == nil {
		return InstanceList{}, ErrRuntimeNotSupported
	}
	discovery, err := s.discovery.Detect(ctx, yorvaruntime.Kind(hermesRuntimeID))
	if err != nil {
		return InstanceList{}, err
	}
	if discovery.State != yorvaruntime.DiscoverySupported || discovery.Selected == nil || discovery.Selected.Path == "" {
		return InstanceList{}, ErrRuntimeNotSupported
	}
	installation, err := s.ensureInstallation(ctx, discovery)
	if err != nil {
		return InstanceList{}, err
	}

	unlock := s.lockInstallation(installation.ID)
	defer unlock()

	now := s.now()
	freshness := "FRESH"
	var queryErr error
	natives, listErr := s.source.List(ctx, discovery.Selected.Path)
	if listErr != nil {
		queryErr = classifyProfileListError(listErr)
		if markErr := s.db.MarkInstancesUnknown(ctx, installation.ID, now); markErr != nil {
			return InstanceList{}, markErr
		}
		freshness = "UNKNOWN"
	} else {
		entries := make([]sqlite.InstanceSnapshotEntry, 0, len(natives))
		for _, native := range natives {
			entries = append(entries, sqlite.InstanceSnapshotEntry{NativeID: native.NativeID, Default: native.Default || native.NativeID == "default"})
		}
		if err := s.db.ApplyInstanceSnapshot(ctx, installation.ID, entries, now); err != nil {
			return InstanceList{}, err
		}
	}

	rows, err := s.db.ListInstances(ctx, installation.ID)
	if err != nil {
		return InstanceList{}, err
	}
	views := make([]InstanceView, 0, len(rows))
	var lastSync *time.Time
	for _, row := range rows {
		views = append(views, instanceView(row))
		if row.LastSyncedAt != nil && (lastSync == nil || row.LastSyncedAt.After(*lastSync)) {
			lastSync = row.LastSyncedAt
		}
	}
	result := InstanceList{
		RuntimeID:             hermesRuntimeID,
		RuntimeInstallationID: installation.ID,
		Freshness:             freshness,
		LastSyncedAt:          lastSync,
		Instances:             views,
		Capabilities:          InstanceCapabilities{Instances: true, Lifecycle: false},
	}
	if queryErr != nil {
		if errors.Is(queryErr, ErrInstanceOutputUnrecognized) {
			result.ErrorCode = yorvaruntime.ErrorInstanceOutputUnrecognized
		} else {
			result.ErrorCode = yorvaruntime.ErrorInstanceQueryFailed
		}
	}
	return result, nil
}

func (s *InstanceInventory) GetInstance(ctx context.Context, instanceID string) (InstanceView, error) {
	if instanceID == "" {
		return InstanceView{}, ErrInstanceNotFound
	}
	row, err := s.db.GetInstance(ctx, instanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return InstanceView{}, ErrInstanceNotFound
	}
	if err != nil {
		return InstanceView{}, err
	}
	return instanceView(row), nil
}

func (s *InstanceInventory) ensureInstallation(ctx context.Context, discovery yorvaruntime.Discovery) (sqlite.AcceptedInstallation, error) {
	now := s.now()
	existing, err := s.db.GetAcceptedInstallation(ctx, s.nodeID, yorvaruntime.Kind(hermesRuntimeID), discovery.Selected.Path)
	if err == nil {
		existing.Version = discovery.Selected.Version
		existing.SupportState = discovery.State
		existing.LastDetectedAt = now
		existing.UpdatedAt = now
		if err := s.db.UpsertAcceptedInstallation(ctx, existing); err != nil {
			return sqlite.AcceptedInstallation{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return sqlite.AcceptedInstallation{}, err
	}
	id, err := sqlite.NewInstallationID()
	if err != nil {
		return sqlite.AcceptedInstallation{}, err
	}
	created := sqlite.AcceptedInstallation{
		ID:             id,
		NodeID:         s.nodeID,
		RuntimeKind:    yorvaruntime.Kind(hermesRuntimeID),
		InstallPath:    discovery.Selected.Path,
		Version:        discovery.Selected.Version,
		SupportState:   discovery.State,
		Status:         "ACCEPTED",
		LastDetectedAt: now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.db.UpsertAcceptedInstallation(ctx, created); err != nil {
		return sqlite.AcceptedInstallation{}, err
	}
	return created, nil
}

func (s *InstanceInventory) lockInstallation(id string) func() {
	s.mu.Lock()
	lock, ok := s.locks[id]
	if !ok {
		lock = &sync.Mutex{}
		s.locks[id] = lock
	}
	s.mu.Unlock()
	lock.Lock()
	return lock.Unlock
}

func instanceView(row instance.Instance) InstanceView {
	return InstanceView{
		InstanceID:            row.ID,
		RuntimeInstallationID: row.RuntimeInstallationID,
		Name:                  row.Name,
		Default:               row.Default,
		Protected:             row.Protected,
		Availability:          row.Availability,
		LastSyncedAt:          row.LastSyncedAt,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
		Capabilities:          InstanceCapabilities{Instances: true, Lifecycle: false},
	}
}

func classifyProfileListError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrInstanceOutputUnrecognized) {
		return ErrInstanceOutputUnrecognized
	}
	return ErrInstanceQueryFailed
}
