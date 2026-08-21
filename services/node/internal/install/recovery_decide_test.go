package install

import (
	"testing"
)

func TestDecideRecoveryMatrix(t *testing.T) {
	const (
		genA = "gen_aaaaaaaaaaaaaaaaaaaaaa"
		genB = "gen_bbbbbbbbbbbbbbbbbbbbbb"
		txnA = "txn_aaaaaaaaaaaaaaaaaaaaaa"
	)
	txn := func(state TransactionState) TransactionView {
		return TransactionView{
			Valid: true, ID: txnA, State: state, GenerationID: genA,
			ActiveBefore: genB, StagingRel: "staging/" + txnA, GenerationRel: "generations/" + genA,
		}
	}
	sealed := TreeObservation{Present: true, Sealed: true, LineageMatch: true, LineageID: genA}
	emptyStaging := TreeObservation{Present: true, Empty: true}
	mutableStaging := TreeObservation{Present: true, Empty: false, LineageMatch: true}
	invalidStaging := TreeObservation{Present: true, Empty: false, Sealed: false}
	pred := ActivePointer{Present: true, Valid: true, GenerationID: genB}
	self := ActivePointer{Present: true, Valid: true, GenerationID: genA}
	unrelated := ActivePointer{Present: true, Valid: true, GenerationID: "gen_cccccccccccccccccccccc"}
	missing := ActivePointer{}
	malformed := ActivePointer{Present: true, Valid: false}
	envOK := EnvironmentObservation{HermesHomeOK: true, PathOK: true}
	envIncomplete := EnvironmentObservation{HermesHomeOK: true, PathOK: false}

	tests := []struct {
		name string
		obs  Observation
		want RecoveryDecision
	}{
		{
			name: "none + unknown dir",
			obs:  Observation{UnknownDirectories: []string{"hermes-agent"}},
			want: RecoveryDecision{Gate: GateReady, Action: ActionDiagnoseRetain},
		},
		{
			name: "none + absent + missing pointer",
			obs:  Observation{Active: missing},
			want: RecoveryDecision{Gate: GateReady, Action: ActionNone},
		},
		{
			name: "none + absent + malformed pointer",
			obs:  Observation{Active: malformed},
			want: RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeBlockedUnsafe},
		},
		{
			name: "none + generations exist + no active never auto-activates",
			obs:  Observation{UnknownDirectories: []string{"generations/gen_orphan"}, Active: missing},
			want: RecoveryDecision{Gate: GateReady, Action: ActionDiagnoseRetain},
		},
		{
			name: "CREATED absent staging and gen",
			obs:  Observation{Transactions: []TransactionView{txn(StateCreated)}},
			want: RecoveryDecision{Gate: GateReady, NextState: StateFailed, Action: ActionNone},
		},
		{
			name: "CREATED empty staging",
			obs:  Observation{Transactions: []TransactionView{txn(StateCreated)}, Staging: emptyStaging},
			want: RecoveryDecision{Gate: GateReady, NextState: StateFailed, Action: ActionRemoveEmptyStaging},
		},
		{
			name: "CREATED nonempty staging",
			obs:  Observation{Transactions: []TransactionView{txn(StateCreated)}, Staging: mutableStaging},
			want: RecoveryDecision{Gate: GateReady, NextState: StateFailed, Action: ActionMoveStagingToFailed},
		},
		{
			name: "CREATED owned candidate is failed without activation",
			obs:  Observation{Transactions: []TransactionView{txn(StateCreated)}, Generation: sealed},
			want: RecoveryDecision{Gate: GateReady, NextState: StateFailed, Action: ActionNone},
		},
		{
			name: "BUILDING empty staging",
			obs:  Observation{Transactions: []TransactionView{txn(StateBuilding)}, Staging: emptyStaging},
			want: RecoveryDecision{Gate: GateReady, NextState: StateFailed, Action: ActionRemoveEmptyStaging, ErrorCode: CodeInterrupted},
		},
		{
			name: "BUILDING mutable staging",
			obs:  Observation{Transactions: []TransactionView{txn(StateBuilding)}, Staging: mutableStaging},
			want: RecoveryDecision{Gate: GateReady, NextState: StateFailed, Action: ActionMoveStagingToFailed, ErrorCode: CodeInterrupted},
		},
		{
			name: "BUILDING absent staging and gen",
			obs:  Observation{Transactions: []TransactionView{txn(StateBuilding)}},
			want: RecoveryDecision{Gate: GateReady, NextState: StateFailed, Action: ActionNone, ErrorCode: CodeInterrupted},
		},
		{
			name: "BUILDING owned candidate is interrupted",
			obs:  Observation{Transactions: []TransactionView{txn(StateBuilding)}, Generation: sealed},
			want: RecoveryDecision{Gate: GateReady, NextState: StateFailed, Action: ActionNone, ErrorCode: CodeInterrupted},
		},
		{
			name: "SEALED legacy staging never publishes",
			obs:  Observation{Transactions: []TransactionView{txn(StateSealed)}, Staging: sealed, Active: pred},
			want: RecoveryDecision{Gate: GateReady, NextState: StateFailed, Action: ActionMoveStagingToFailed, ErrorCode: CodeInterrupted},
		},
		{
			name: "SEALED infer rename complete",
			obs:  Observation{Transactions: []TransactionView{txn(StateSealed)}, Generation: sealed, Active: pred},
			want: RecoveryDecision{Gate: GateReconciling, NextState: StatePublished, Action: ActionPersistPublished},
		},
		{
			name: "SEALED invalid staging",
			obs:  Observation{Transactions: []TransactionView{txn(StateSealed)}, Staging: invalidStaging},
			want: RecoveryDecision{Gate: GateReady, NextState: StateFailed, Action: ActionMoveStagingToFailed, ErrorCode: CodeSealInvalid},
		},
		{
			name: "SEALED leftover empty staging after rename",
			obs:  Observation{Transactions: []TransactionView{txn(StateSealed)}, Staging: emptyStaging, Generation: sealed},
			want: RecoveryDecision{Gate: GateReconciling, NextState: StatePublished, Action: ActionRemoveEmptyStaging},
		},
		{
			name: "SEALED duplicate complete trees",
			obs: Observation{
				Transactions: []TransactionView{txn(StateSealed)},
				Staging:      sealed,
				Generation:   sealed,
			},
			want: RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeBlockedUnsafe},
		},
		{
			name: "SEALED different lineage",
			obs: Observation{
				Transactions: []TransactionView{txn(StateSealed)},
				Staging:      sealed,
				Generation:   TreeObservation{Present: true, Sealed: true, LineageID: genB},
			},
			want: RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeBlockedUnsafe},
		},
		{
			name: "PUBLISHED activate forward",
			obs:  Observation{Transactions: []TransactionView{txn(StatePublished)}, Generation: sealed, Active: pred},
			want: RecoveryDecision{Gate: GateReconciling, NextState: StateActivating, Action: ActionPersistActivating},
		},
		{
			name: "PUBLISHED already active",
			obs:  Observation{Transactions: []TransactionView{txn(StatePublished)}, Generation: sealed, Active: self},
			want: RecoveryDecision{Gate: GateReconciling, NextState: StateActivating, Action: ActionReconcileEnv},
		},
		{
			name: "PUBLISHED missing generation",
			obs:  Observation{Transactions: []TransactionView{txn(StatePublished)}, Active: pred},
			want: RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeInconsistent},
		},
		{
			name: "PUBLISHED leftover staging",
			obs: Observation{
				Transactions: []TransactionView{txn(StatePublished)},
				Staging:      mutableStaging,
				Generation:   sealed,
				Active:       pred,
			},
			want: RecoveryDecision{Gate: GateReconciling, Action: ActionGCStaging},
		},
		{
			name: "PUBLISHED leftover empty staging",
			obs: Observation{
				Transactions: []TransactionView{txn(StatePublished)},
				Staging:      emptyStaging,
				Generation:   sealed,
				Active:       pred,
			},
			want: RecoveryDecision{Gate: GateReconciling, Action: ActionRemoveEmptyStaging},
		},
		{
			name: "PUBLISHED unrelated pointer",
			obs: Observation{
				Transactions: []TransactionView{txn(StatePublished)},
				Generation:   sealed,
				Active:       unrelated,
			},
			want: RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeBlockedUnsafe},
		},
		{
			name: "PUBLISHED missing pointer first install",
			obs: Observation{
				Transactions: []TransactionView{{
					Valid: true, ID: txnA, State: StatePublished, GenerationID: genA,
					ActiveBeforeKind: ActiveBeforeAbsent,
					StagingRel:       "staging/" + txnA, GenerationRel: "generations/" + genA,
				}},
				Generation: sealed,
				Active:     missing,
			},
			want: RecoveryDecision{Gate: GateReconciling, NextState: StateActivating, Action: ActionPersistActivating},
		},
		{
			name: "PUBLISHED missing pointer after recorded predecessor",
			obs:  Observation{Transactions: []TransactionView{txn(StatePublished)}, Generation: sealed, Active: missing},
			want: RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeInconsistent},
		},
		{
			name: "PUBLISHED predecessor digest mismatch",
			obs: Observation{
				Transactions: []TransactionView{{
					Valid: true, ID: txnA, State: StatePublished, GenerationID: genA,
					ActiveBefore: genB, ActiveBeforeDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					ActiveBeforeKind: ActiveBeforeValid,
					StagingRel:       "staging/" + txnA, GenerationRel: "generations/" + genA,
				}},
				Generation: sealed,
				Active: ActivePointer{
					Class: ActiveValid, Present: true, Valid: true, GenerationID: genB,
					SealSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				},
			},
			want: RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeBlockedUnsafe},
		},
		{
			name: "ACTIVATING complete pointer",
			obs:  Observation{Transactions: []TransactionView{txn(StateActivating)}, Generation: sealed, Active: pred},
			want: RecoveryDecision{Gate: GateReconciling, Action: ActionActivate},
		},
		{
			name: "ACTIVATING already new env incomplete",
			obs: Observation{
				Transactions: []TransactionView{txn(StateActivating)},
				Generation:   sealed,
				Active:       self,
				Environment:  envIncomplete,
			},
			want: RecoveryDecision{Gate: GateReconciling, Action: ActionReconcileEnv},
		},
		{
			name: "ACTIVATING already new env complete",
			obs: Observation{
				Transactions: []TransactionView{txn(StateActivating)},
				Generation:   sealed,
				Active:       self,
				Environment:  envOK,
			},
			want: RecoveryDecision{Gate: GateReconciling, NextState: StateCommitted, Action: ActionCommit},
		},
		{
			name: "ACTIVATING unrelated pointer",
			obs: Observation{
				Transactions: []TransactionView{txn(StateActivating)},
				Generation:   sealed,
				Active:       unrelated,
			},
			want: RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeBlockedUnsafe},
		},
		{
			name: "ACTIVATING invalid generation",
			obs:  Observation{Transactions: []TransactionView{txn(StateActivating)}, Active: pred},
			want: RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeInconsistent},
		},
		{
			name: "COMMITTED active this gen",
			obs: Observation{
				Transactions: []TransactionView{txn(StateCommitted)},
				Generation:   sealed,
				Active:       self,
				Environment:  envOK,
			},
			want: RecoveryDecision{Gate: GateReady, Action: ActionNone},
		},
		{
			name: "COMMITTED leftover staging",
			obs: Observation{
				Transactions: []TransactionView{txn(StateCommitted)},
				Staging:      mutableStaging,
				Generation:   sealed,
				Active:       self,
				Environment:  envOK,
			},
			want: RecoveryDecision{Gate: GateReady, Action: ActionGCStaging},
		},
		{
			name: "COMMITTED env incomplete optional reconcile",
			obs: Observation{
				Transactions: []TransactionView{txn(StateCommitted)},
				Generation:   sealed,
				Active:       self,
				Environment:  envIncomplete,
			},
			want: RecoveryDecision{Gate: GateReady, Action: ActionReconcileEnv},
		},
		{
			name: "FAILED not active never resume",
			obs: Observation{
				Transactions: []TransactionView{txn(StateFailed)},
				Staging:      mutableStaging,
				Active:       pred,
			},
			want: RecoveryDecision{Gate: GateReady, Action: ActionNone},
		},
		{
			name: "FAILED but active.json names this gen",
			obs: Observation{
				Transactions: []TransactionView{txn(StateFailed)},
				Generation:   sealed,
				Active:       self,
				Environment:  envOK,
			},
			want: RecoveryDecision{Gate: GateReady, NextState: StateCommitted, Action: ActionCommit},
		},
		{
			name: "FAILED but active env incomplete rolls to ACTIVATING",
			obs: Observation{
				Transactions: []TransactionView{txn(StateFailed)},
				Generation:   sealed,
				Active:       self,
				Environment:  envIncomplete,
			},
			want: RecoveryDecision{Gate: GateReconciling, NextState: StateActivating, Action: ActionReconcileEnv},
		},
		{
			name: "malformed txn without reserved name is ignored",
			obs:  Observation{Transactions: []TransactionView{{ID: "scratch"}}},
			want: RecoveryDecision{Gate: GateReady, Action: ActionNone},
		},
		{
			name: "malformed reserved name",
			obs: Observation{Transactions: []TransactionView{{
				OccupiesReservedName: true, ID: "txn_bad",
			}}},
			want: RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeBlockedUnsafe},
		},
		{
			name: "two failable nonterminal recover each once",
			obs: Observation{Transactions: []TransactionView{
				txn(StateBuilding),
				{Valid: true, ID: "txn_other", State: StateCreated, GenerationID: genB},
			}},
			want: RecoveryDecision{Gate: GateReconciling, Action: ActionFailFailableExtras, ErrorCode: CodeInterrupted},
		},
		{
			name: "two sealed nonterminal still blocked",
			obs: Observation{Transactions: []TransactionView{
				txn(StateSealed),
				{Valid: true, ID: "txn_other", State: StatePublished, GenerationID: genB},
			}},
			want: RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeBlockedUnsafe},
		},
		{
			name: "reserved id collision",
			obs:  Observation{ReservedIDCollision: true},
			want: RecoveryDecision{Gate: GateBlockedUnsafe, Action: ActionBlockUnsafe, ErrorCode: CodeBlockedUnsafe},
		},
		{
			name: "history does not compete with other active",
			obs: Observation{
				Transactions: []TransactionView{txn(StateCommitted)},
				Active:       unrelated,
			},
			want: RecoveryDecision{Gate: GateReady, Action: ActionNone},
		},
		{
			name: "D4 unknown directory never scheduled for delete",
			obs:  Observation{UnknownDirectories: []string{"mystery", "hermes-agent"}},
			want: RecoveryDecision{Gate: GateReady, Action: ActionDiagnoseRetain},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DecideRecovery(tc.obs)
			if got != tc.want {
				t.Fatalf("DecideRecovery() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestDecideRecoveryIdempotentOrForward(t *testing.T) {
	const genA = "gen_aaaaaaaaaaaaaaaaaaaaaa"
	pred := ActivePointer{Present: true, Valid: true, GenerationID: "gen_bbbbbbbbbbbbbbbbbbbbbb"}
	sealed := TreeObservation{Present: true, Sealed: true, LineageMatch: true, LineageID: genA}
	txn := func(state TransactionState) TransactionView {
		return TransactionView{
			Valid: true, ID: "txn_aaaaaaaaaaaaaaaaaaaaaa", State: state,
			GenerationID: genA, ActiveBefore: pred.GenerationID,
		}
	}
	cases := []struct {
		name string
		obs  Observation
	}{
		{
			name: "ACTIVATING env incomplete",
			obs: Observation{
				Transactions: []TransactionView{txn(StateActivating)},
				Generation:   sealed,
				Active:       ActivePointer{Present: true, Valid: true, GenerationID: genA},
				Environment:  EnvironmentObservation{HermesHomeOK: true, PathOK: false},
			},
		},
		{
			name: "PUBLISHED leftover staging",
			obs: Observation{
				Transactions: []TransactionView{txn(StatePublished)},
				Staging:      TreeObservation{Present: true, Empty: false, LineageMatch: true},
				Generation:   sealed,
				Active:       pred,
			},
		},
		{
			name: "SEALED final generation",
			obs: Observation{
				Transactions: []TransactionView{txn(StateSealed)},
				Generation:   sealed,
				Active:       pred,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d1 := DecideRecovery(tc.obs)
			if d1.Gate == GateBlockedUnsafe {
				t.Fatalf("D1 blocked: %#v", d1)
			}
			obs2 := applyDecision(tc.obs, d1)
			d2 := DecideRecovery(obs2)
			if d2.Gate == GateBlockedUnsafe {
				t.Fatalf("D2 blocked after apply: D1=%#v D2=%#v", d1, d2)
			}
			if d2 == d1 && d1.Action != ActionNone {
				t.Fatalf("D2 repeated mutating decision: %#v", d2)
			}
			obs3 := applyDecision(obs2, d2)
			d3 := DecideRecovery(obs3)
			if d3.Gate == GateBlockedUnsafe {
				t.Fatalf("D3 blocked: %#v", d3)
			}
			if d3 == d2 && d2.Action != ActionNone {
				t.Fatalf("D3 repeated mutating decision: %#v", d3)
			}
		})
	}
}

func applyDecision(obs Observation, d RecoveryDecision) Observation {
	if n := len(obs.Transactions); n > 0 {
		copied := make([]TransactionView, n)
		copy(copied, obs.Transactions)
		obs.Transactions = copied
	}
	if d.NextState != "" && len(obs.Transactions) == 1 && obs.Transactions[0].Valid {
		obs.Transactions[0].State = d.NextState
	}
	switch d.Action {
	case ActionRemoveEmptyStaging, ActionMoveStagingToFailed, ActionGCStaging:
		obs.Staging = TreeObservation{}
	case ActionRemoveEmptyGeneration:
		obs.Generation = TreeObservation{}
	case ActionActivate:
		if len(obs.Transactions) == 1 {
			obs.Active = ActivePointer{Present: true, Valid: true, GenerationID: obs.Transactions[0].GenerationID}
		}
	case ActionReconcileEnv:
		obs.Environment = EnvironmentObservation{HermesHomeOK: true, PathOK: true}
	case ActionCommit:
		if len(obs.Transactions) == 1 {
			obs.Transactions[0].State = StateCommitted
		}
	}
	return obs
}
