package install

import (
	"context"
	"errors"
	"sync"
)

const maxRecoverSteps = 8

// GateHolder is the daemon install-subsystem admission gate.
type GateHolder struct {
	mu   sync.RWMutex
	gate Gate
}

func NewGateHolder() *GateHolder {
	return &GateHolder{gate: GateReconciling}
}

func (g *GateHolder) Get() Gate {
	if g == nil {
		return GateReady
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.gate == "" {
		return GateReady
	}
	return g.gate
}

func (g *GateHolder) Set(gate Gate) {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.gate = gate
	g.mu.Unlock()
}

func Recover(ctx context.Context, root string, gate *GateHolder) (RecoveryDecision, error) {
	return RecoverWith(ctx, root, gate, defaultEnvironmentStore())
}

func RecoverWith(ctx context.Context, root string, gate *GateHolder, env EnvironmentStore) (RecoveryDecision, error) {
	gate.Set(GateReconciling)
	lock, err := AcquireLock(root)
	if err != nil {
		if errors.Is(err, ErrLockBusy) {
			gate.Set(GateReconciling)
			return RecoveryDecision{Gate: GateReconciling, Action: ActionNone, ErrorCode: CodeNotReady}, err
		}
		gate.Set(GateBlockedUnsafe)
		return RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeBlockedUnsafe}, err
	}
	defer lock.Release()

	store, err := NewStore(root)
	if err != nil {
		gate.Set(GateBlockedUnsafe)
		return RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeBlockedUnsafe}, err
	}
	mgr := NewManager(store, nil, nil).withEnv(env)

	var previous RecoveryDecision
	for step := 0; step < maxRecoverSteps; step++ {
		obs, err := Observe(store, env)
		if err != nil {
			gate.Set(GateBlockedUnsafe)
			return RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeBlockedUnsafe}, err
		}
		decision := DecideRecovery(obs)
		if decision.Gate == GateBlockedUnsafe || decision.Action == ActionBlockUnsafe {
			gate.Set(GateBlockedUnsafe)
			return decision, nil
		}
		if (decision.Action == ActionNone && decision.NextState == "") || decision.Action == ActionDiagnoseRetain {
			gate.Set(finalGate(decision))
			if gate.Get() == GateReady {
				_, _ = Collect(store, GCHooks{})
			}
			return decision, nil
		}
		if step > 0 && !forwardOrNoop(previous, decision) {
			gate.Set(GateBlockedUnsafe)
			return decision, ErrBlockedUnsafe
		}
		if err := Execute(ctx, store, mgr, obs, decision); err != nil {
			if decision.Action == ActionReconcileEnv {
				gate.Set(GateReconciling)
				return decision, err
			}
			gate.Set(GateBlockedUnsafe)
			return decision, err
		}
		previous = decision
	}
	gate.Set(GateBlockedUnsafe)
	return RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeBlockedUnsafe}, ErrBlockedUnsafe
}

func finalGate(d RecoveryDecision) Gate {
	if d.Gate == GateBlockedUnsafe {
		return GateBlockedUnsafe
	}
	if d.Gate == GateReconciling {
		return GateReconciling
	}
	return GateReady
}

func forwardOrNoop(prev, next RecoveryDecision) bool {
	if next.Gate == GateBlockedUnsafe || next.Action == ActionBlockUnsafe {
		return false
	}
	if next.Action == ActionNone || next.Action == ActionDiagnoseRetain {
		return true
	}
	if next.Action == ActionReconcileEnv {
		return true
	}
	if prev.Action != next.Action {
		return true
	}
	if prev.NextState != "" && next.NextState != "" && prev.NextState != next.NextState {
		return rankState(next.NextState) > rankState(prev.NextState)
	}
	return false
}

func rankState(state TransactionState) int {
	switch state {
	case StateCreated:
		return 1
	case StateBuilding:
		return 2
	case StateSealed:
		return 3
	case StatePublished:
		return 4
	case StateActivating:
		return 5
	case StateCommitted:
		return 6
	case StateFailed:
		return 0
	default:
		return 0
	}
}
