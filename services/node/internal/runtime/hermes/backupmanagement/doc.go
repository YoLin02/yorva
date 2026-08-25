// Package backupmanagement provides side-effect-free structural verification
// for the proposed Hermes Runtime backup container.
//
// It deliberately does not encrypt, decrypt, extract, write files, invoke
// Hermes, or authorize Restore mutation. The package is an unconnected safety
// core while ADR-0013 remains proposed.
package backupmanagement
