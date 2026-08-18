package install

import (
	"strings"
	"testing"
)

func TestClosedIDs(t *testing.T) {
	txnID, err := NewTransactionID()
	if err != nil {
		t.Fatal(err)
	}
	genID, err := NewGenerationID()
	if err != nil {
		t.Fatal(err)
	}
	if err := ParseTransactionID(txnID); err != nil {
		t.Fatalf("txn: %v", err)
	}
	if err := ParseGenerationID(genID); err != nil {
		t.Fatalf("gen: %v", err)
	}
	if ParseTransactionID(genID) == nil || ParseGenerationID(txnID) == nil {
		t.Fatal("prefixes must not be interchangeable")
	}
	if ParseTransactionID("txn_bad") == nil || ParseGenerationID("gen_"+strings.Repeat("A", 22)) == nil {
		t.Fatal("short or uppercase ids must be rejected")
	}
	if ParseTransactionID("txn_"+strings.Repeat("a", 22)+"/../x") == nil {
		t.Fatal("id with path bytes must be rejected")
	}
	if !OccupiesReservedTxnName("txn_bad") || !OccupiesReservedGenName("gen_orphan") {
		t.Fatal("reserved prefix occupancy")
	}
	if OccupiesReservedTxnName("scratch") {
		t.Fatal("non-reserved name")
	}
}
