package install

// DecideRecovery is a pure function of observed transaction, filesystem and
// active.json bytes. It never reads SQLite Operations, PATH as activation,
// or "newest generation" as a pointer.
func DecideRecovery(obs Observation) RecoveryDecision {
	if obs.ReservedIDCollision {
		return blocked("reserved identifier collision")
	}
	var valid, nonterminal []TransactionView
	for _, txn := range obs.Transactions {
		if !txn.Valid {
			if txn.OccupiesReservedName {
				return blocked("malformed transaction occupies a reserved name")
			}
			continue
		}
		valid = append(valid, txn)
		if txn.State.Nonterminal() {
			nonterminal = append(nonterminal, txn)
		}
	}
	if len(nonterminal) > 1 {
		return blocked("multiple nonterminal transactions")
	}
	if len(nonterminal) == 0 {
		return decideTerminalWorld(obs, valid)
	}
	return decideOne(obs, nonterminal[0])
}

func decideTerminalWorld(obs Observation, valid []TransactionView) RecoveryDecision {
	if pointerUnrelatedToHistory(obs, valid) {
		return RecoveryDecision{Gate: GateReady, Action: ActionNone}
	}
	if _, ok := failedButActive(obs, valid); ok {
		if obs.Environment.Complete() {
			return RecoveryDecision{Gate: GateReady, NextState: StateCommitted, Action: ActionCommit}
		}
		return RecoveryDecision{Gate: GateReconciling, NextState: StateActivating, Action: ActionReconcileEnv}
	}
	if _, ok := committedActive(obs, valid); ok {
		if action, ok := leftoverStaging(obs); ok {
			return RecoveryDecision{Gate: GateReady, Action: action}
		}
		if !obs.Environment.Complete() && obs.Active.Valid {
			return RecoveryDecision{Gate: GateReady, Action: ActionReconcileEnv}
		}
		return RecoveryDecision{Gate: GateReady, Action: ActionNone}
	}
	if len(obs.UnknownDirectories) > 0 {
		return RecoveryDecision{Gate: GateReady, Action: ActionDiagnoseRetain}
	}
	return RecoveryDecision{Gate: GateReady, Action: ActionNone}
}

func decideOne(obs Observation, txn TransactionView) RecoveryDecision {
	switch txn.State {
	case StateCreated:
		return decideCreated(obs)
	case StateBuilding:
		return decideBuilding(obs)
	case StateSealed:
		return decideSealed(obs)
	case StatePublished:
		return decidePublished(obs, txn)
	case StateActivating:
		return decideActivating(obs, txn)
	default:
		return blocked("unknown nonterminal state")
	}
}

func decideCreated(obs Observation) RecoveryDecision {
	if !obs.Staging.Present && obs.Generation.Present && (obs.Generation.LineageMatch || obs.Generation.Sealed) {
		return failClosed(CodeInconsistent)
	}
	if !obs.Staging.Present && !obs.Generation.Present {
		return failTxn("", ActionNone)
	}
	if obs.Staging.Present && !obs.Generation.Present {
		if obs.Staging.Empty {
			return failTxn("", ActionRemoveEmptyStaging)
		}
		return failTxn("", ActionMoveStagingToFailed)
	}
	return failClosed(CodeInconsistent)
}

func decideBuilding(obs Observation) RecoveryDecision {
	if !obs.Staging.Present && obs.Generation.Present && obs.Generation.LineageMatch {
		return failClosed(CodeInconsistent)
	}
	if !obs.Staging.Present && !obs.Generation.Present {
		return failTxn(CodeInterrupted, ActionNone)
	}
	if obs.Staging.Present && !obs.Generation.Present {
		if obs.Staging.Empty {
			return failTxn(CodeInterrupted, ActionRemoveEmptyStaging)
		}
		return failTxn(CodeInterrupted, ActionMoveStagingToFailed)
	}
	return failClosed(CodeInconsistent)
}

func decideSealed(obs Observation) RecoveryDecision {
	if obs.Generation.Present && obs.Generation.Sealed && obs.Generation.LineageMatch && obs.Staging.Present && !obs.Staging.Empty && obs.Staging.Sealed && obs.Staging.LineageMatch {
		return blocked("sealed staging and generation both complete")
	}
	if obs.Generation.Present && obs.Generation.LineageID != "" && !obs.Generation.LineageMatch {
		return blocked("generation lineage does not match transaction")
	}
	if !obs.Staging.Present && obs.Generation.Present && obs.Generation.Sealed && obs.Generation.LineageMatch {
		return RecoveryDecision{Gate: GateReconciling, NextState: StatePublished, Action: ActionNone}
	}
	if obs.Staging.Present && obs.Staging.Empty && obs.Generation.Present && obs.Generation.Sealed && obs.Generation.LineageMatch {
		return RecoveryDecision{Gate: GateReconciling, NextState: StatePublished, Action: ActionRemoveEmptyStaging}
	}
	if obs.Staging.Present && obs.Staging.Sealed && obs.Staging.LineageMatch && !obs.Generation.Present {
		return RecoveryDecision{Gate: GateReconciling, NextState: StatePublished, Action: ActionPublish}
	}
	if obs.Staging.Present && !obs.Staging.Sealed && !obs.Generation.Present {
		return failTxn(CodeSealInvalid, ActionMoveStagingToFailed)
	}
	return failClosed(CodeInconsistent)
}

func decidePublished(obs Observation, txn TransactionView) RecoveryDecision {
	if !obs.Generation.Present || !obs.Generation.Sealed || !obs.Generation.LineageMatch {
		return failClosed(CodeInconsistent)
	}
	if action, ok := leftoverStaging(obs); ok {
		return RecoveryDecision{Gate: GateReconciling, Action: action}
	}
	if activeNames(obs, txn.GenerationID) {
		return RecoveryDecision{Gate: GateReconciling, NextState: StateActivating, Action: ActionReconcileEnv}
	}
	if activeIsPredecessorOrNone(obs, txn) {
		return RecoveryDecision{Gate: GateReconciling, NextState: StateActivating, Action: ActionPersistActivating}
	}
	if obs.Active.Valid {
		return blocked("published generation but active.json names an unrelated generation")
	}
	return failClosed(CodeInconsistent)
}

func decideActivating(obs Observation, txn TransactionView) RecoveryDecision {
	if !obs.Generation.Present || !obs.Generation.Sealed || !obs.Generation.LineageMatch {
		return failClosed(CodeInconsistent)
	}
	if obs.Active.Valid && !activeNames(obs, txn.GenerationID) && !activeIsPredecessorOrNone(obs, txn) {
		return blocked("activating while active.json names an unrelated generation")
	}
	if activeNames(obs, txn.GenerationID) {
		if obs.Environment.Complete() {
			return RecoveryDecision{Gate: GateReconciling, NextState: StateCommitted, Action: ActionCommit}
		}
		return RecoveryDecision{Gate: GateReconciling, Action: ActionReconcileEnv}
	}
	if activeIsPredecessorOrNone(obs, txn) {
		return RecoveryDecision{Gate: GateReconciling, Action: ActionActivate}
	}
	return failClosed(CodeInconsistent)
}

func leftoverStaging(obs Observation) (Action, bool) {
	if !obs.Staging.Present || !obs.Generation.Present || !obs.Generation.Sealed || !obs.Generation.LineageMatch {
		return "", false
	}
	if obs.Staging.Empty {
		return ActionRemoveEmptyStaging, true
	}
	if obs.Staging.LineageMatch {
		return ActionGCStaging, true
	}
	return "", false
}

func activeNames(obs Observation, generationID string) bool {
	return obs.Active.Valid && generationID != "" && obs.Active.GenerationID == generationID
}

func activeIsPredecessorOrNone(obs Observation, txn TransactionView) bool {
	if !obs.Active.Present || !obs.Active.Valid {
		return true
	}
	if obs.Active.GenerationID == txn.GenerationID {
		return false
	}
	if txn.ActiveBefore != "" {
		return obs.Active.GenerationID == txn.ActiveBefore
	}
	return true
}

func committedActive(obs Observation, valid []TransactionView) (TransactionView, bool) {
	for _, txn := range valid {
		if txn.State == StateCommitted && obs.Active.Valid && obs.Active.GenerationID == txn.GenerationID {
			return txn, true
		}
	}
	return TransactionView{}, false
}

func failedButActive(obs Observation, valid []TransactionView) (TransactionView, bool) {
	for _, txn := range valid {
		if txn.State == StateFailed && obs.Active.Valid && obs.Active.GenerationID == txn.GenerationID {
			return txn, true
		}
	}
	return TransactionView{}, false
}

func pointerUnrelatedToHistory(obs Observation, valid []TransactionView) bool {
	if !obs.Active.Valid {
		return false
	}
	for _, txn := range valid {
		if txn.GenerationID == obs.Active.GenerationID {
			return false
		}
	}
	return len(valid) > 0
}

func failTxn(code string, action Action) RecoveryDecision {
	return RecoveryDecision{Gate: GateReady, NextState: StateFailed, Action: action, ErrorCode: code}
}

func failClosed(code string) RecoveryDecision {
	return RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: code}
}

func blocked(string) RecoveryDecision {
	return RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeBlockedUnsafe}
}
