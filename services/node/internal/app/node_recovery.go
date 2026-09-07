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
	if s.discovery == nil || s.db == nil || s.source == nil {
		return NodeRecovery{}, ErrRuntimeNotSupported
	}
	detected, err := s.discovery.Detect(ctx, yorvaruntime.Kind(hermesRuntimeID))
	if err != nil {
		return NodeRecovery{ErrorCode: yorvaruntime.ErrorInstanceQueryFailed}, nil
	}
	if detected.State == yorvaruntime.DiscoveryNotInstalled {
		// A fresh YORVA installation can update before a Runtime is installed. A
		// missing previously accepted Runtime cannot silently pass recovery.
		count, err := s.db.CountAcceptedInstallations(ctx)
		if err != nil {
			return NodeRecovery{}, err
		}
		if count == 0 {
			return NodeRecovery{Ready: true}, nil
		}
		return NodeRecovery{ErrorCode: yorvaruntime.ErrorRuntimeNotInstalled}, nil
	}
	if detected.State != yorvaruntime.DiscoverySupported {
		code := detected.ErrorCode
		if code == "" {
			code = yorvaruntime.ErrorRuntimeNotSupported
		}
		return NodeRecovery{ErrorCode: code}, nil
	}
	listed, err := s.ListInstances(ctx, hermesRuntimeID)
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
	return NodeRecovery{Ready: true}, nil
}
