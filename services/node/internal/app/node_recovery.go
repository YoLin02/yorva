package app

import (
	"context"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

// NodeRecovery separates a usable management daemon from verified Runtime recovery.
// It is evaluated after startup migration and journal recovery, using live readback.
type NodeRecovery struct {
	Ready     bool
	ErrorCode yorvaruntime.ErrorCode
}

func (s *InstanceInventory) CheckNodeRecovery(ctx context.Context) (NodeRecovery, error) {
	if s.discovery == nil || s.db == nil || s.discovery.registry == nil {
		return NodeRecovery{}, ErrRuntimeNotSupported
	}
	acceptedIDs, err := s.db.ListAcceptedInstallationIDs(ctx, s.nodeID)
	if err != nil {
		return NodeRecovery{}, err
	}
	observed := make(map[string]bool)
	for _, kind := range s.discovery.registry.Kinds() {
		detected, err := s.discovery.Detect(ctx, kind)
		if err != nil {
			return NodeRecovery{ErrorCode: yorvaruntime.ErrorInstanceQueryFailed}, nil
		}
		if detected.State == yorvaruntime.DiscoveryNotInstalled {
			continue
		}
		if detected.State != yorvaruntime.DiscoverySupported {
			code := detected.ErrorCode
			if code == "" {
				code = yorvaruntime.ErrorRuntimeNotSupported
			}
			return NodeRecovery{ErrorCode: code}, nil
		}
		listed, err := s.ListInstances(ctx, string(kind))
		if err != nil {
			return NodeRecovery{ErrorCode: errorCodeFrom(err)}, nil
		}
		if listed.ErrorCode != "" {
			return NodeRecovery{ErrorCode: listed.ErrorCode}, nil
		}
		if listed.Freshness != "FRESH" {
			return NodeRecovery{ErrorCode: yorvaruntime.ErrorInstanceQueryFailed}, nil
		}
		for _, item := range listed.Instances {
			if item.Availability == instance.Unknown {
				return NodeRecovery{ErrorCode: yorvaruntime.ErrorInstanceQueryFailed}, nil
			}
		}
		observed[listed.RuntimeInstallationID] = true
	}
	for _, id := range acceptedIDs {
		if !observed[id] {
			return NodeRecovery{ErrorCode: yorvaruntime.ErrorRuntimeNotInstalled}, nil
		}
	}
	return NodeRecovery{Ready: true}, nil
}
