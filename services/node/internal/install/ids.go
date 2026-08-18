package install

import (
	"crypto/rand"
	"fmt"
	"strings"
)

const (
	idAlphabet    = "abcdefghijklmnopqrstuvwxyz234567"
	idBodyLen     = 22
	txnPrefix     = "txn_"
	genPrefix     = "gen_"
	txnIDLen      = len(txnPrefix) + idBodyLen
	genIDLen      = len(genPrefix) + idBodyLen
	txnFileSuffix = ".json"
)

func NewTransactionID() (string, error) {
	return newClosedID(txnPrefix)
}

func NewGenerationID() (string, error) {
	return newClosedID(genPrefix)
}

func ParseTransactionID(id string) error {
	return parseClosedID(id, txnPrefix)
}

func ParseGenerationID(id string) error {
	return parseClosedID(id, genPrefix)
}

func OccupiesReservedTxnName(name string) bool {
	return strings.HasPrefix(name, txnPrefix)
}

func OccupiesReservedGenName(name string) bool {
	return strings.HasPrefix(name, genPrefix)
}

func transactionFileName(id string) string {
	return id + txnFileSuffix
}

func newClosedID(prefix string) (string, error) {
	var body [idBodyLen]byte
	if _, err := rand.Read(body[:]); err != nil {
		return "", fmt.Errorf("install: generate id: %w", err)
	}
	for i := range body {
		body[i] = idAlphabet[int(body[i])%len(idAlphabet)]
	}
	return prefix + string(body[:]), nil
}

func parseClosedID(id, prefix string) error {
	if len(id) != len(prefix)+idBodyLen || !strings.HasPrefix(id, prefix) {
		return ErrInvalidID
	}
	for i := len(prefix); i < len(id); i++ {
		if !strings.ContainsRune(idAlphabet, rune(id[i])) {
			return ErrInvalidID
		}
	}
	return nil
}
