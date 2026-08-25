// Package backupmanagement provides age v1 encryption and authenticated,
// side-effect-free structural verification for the Hermes Runtime backup
// container.
//
// It deliberately does not select paths, persist identities or passphrases,
// extract files, invoke Hermes, wire a production capability, or authorize
// Restore mutation. Callers own destination publication and secure temporary
// storage; this package only accepts in-memory X25519 identities or
// passphrases and fixed reader/writer boundaries.
package backupmanagement
