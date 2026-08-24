# Phase 6 Owner-Authenticated Weixin and WeCom Smoke

- Attestation received: 2026-08-24 (Asia/Shanghai)
- Evidence classification: **OWNER-ATTESTED EXACT-MSI SUPPLEMENT**
- Exact-candidate classification: **ESTABLISHED FOR THE REMEDIATION MSI**
- Redaction status: **SANITIZED; NO QR, SECRET, PAIRING CODE OR ACCOUNT VALUE**

## Owner attestation

The Repository Owner confirmed that validation with real Weixin and real WeCom
credentials/accounts passed. This record preserves that statement as Owner-provided
input. It does not add unreported test steps, account identity, message content or build
identity.

On 2026-08-24 the Owner supplied the following additional sanitized confirmation:

> Weixin connection, pairing and disconnect, and WeCom connection and disconnect all
> passed on the MSI identified below. No QR, Secret, pairing-code or account-information
> disclosure was observed.

The quotation is an English rendering of the Owner's Chinese confirmation. No sensitive
value was supplied with it. This supplement establishes the tested package and the named
flow outcomes; it does not manufacture a raw transcript, screenshot or account identity.

## Observed host and Runtime facts

Read-only inspection on 2026-08-24 established:

- Windows 11 Pro, version `10.0.26200`, build `26200`, x64;
- YORVA's local node projection reports `0.0.0-dev` on Windows/amd64;
- Hermes is detected as `0.20.2`, `SUPPORTED` and `ACCEPTED`;
- the local database contains both a default and named available Profile, but the Owner
  did not identify which Profile was used for each real-account validation;
- SQLite integrity check returned `ok`.

No QR image/payload, Bot ID, Secret, token, pairing code, cookie, sender/account ID,
Profile name or screenshot is included here.

## Build and MSI mapping

The supplemental confirmation identifies this inspected remediation package:

- file: `Yorva_0.3.2_x64_en-US.msi`;
- size: 120,213,504 bytes;
- SHA-256: `B942E637BE9BC59D6C5B603DA117C7AC9646CD254E88F36B1A609A0696FA8EFD`;
- package result: pinned six-input preparation and MSI inventory inspection passed;
- product inputs: remediation code `a9d903364d7e1403d649895d77367a5806be1b0c`
  through clean product checkout `5009781445c4db8c22d9073b9973ad90c9dd7c5f`;
- audited evidence candidate: `7e1123e216528ae1caf9818b6e0e32b173eeb4d1`;
  its successors after the product checkout changed documentation/evidence only.

The original broad attestation predated this mapping and was correctly classified as
product-level evidence at that time. The later Owner supplement binds the exercised
results to the inspected remediation MSI; it does not claim equivalence with unrelated
packages merely because they have the same filename/version.

Two packages were distinguished during closeout:

- exact product candidate `276991b`: CI artifact
  `Yorva_0.3.2_x64_en-US.msi`, SHA-256
  `5333022B304692E4C5BAAFC03FA5F666E9BB8483F240E2CDD608E9F1DE48EA28`, from Windows
  MSI run `32459991229`; no evidence says the Owner tested this package;
- a different local `Yorva_0.3.2_x64_en-US.msi`, SHA-256
  `8432F09721F4A44ACBD0C086A6CD4D8AE933731C04019BD789F3488F16E52261`, whose inventory
  includes unrelated embedded-Python work and is not the Phase 6 candidate. It is not
  used as Phase 6 package evidence.

The package hashes differ. No equivalence is inferred from the shared filename/version.

## Exercised-result matrix

| Check | Recorded result | Evidence boundary |
| --- | --- | --- |
| Real Weixin validation | **PASS — Owner attested** | Exact remediation MSI identified above. |
| Weixin QR generated/scanned/confirmed | **PASS — Owner-attested connection flow** | The Owner confirmed that the real Weixin connection passed on the exact MSI; no QR bytes or image are retained. |
| Real message received an AI response | **Not evidenced** | Owner statement did not include a message-response check. |
| Sender pairing request/Desktop approval | **PASS — Owner attested** | Exact remediation MSI; no pairing code, sender or account value retained. |
| Local Weixin disconnect | **PASS — Owner attested** | Exact remediation MSI; local disconnect only, with no remote-revocation claim. |
| Real WeCom typed Bot ID/Secret validation | **PASS — Owner attested connection flow** | Exact remediation MSI and the only approved WeCom connect path is the typed manual flow; no credential value retained. |
| WeCom reached `CONNECTED` | **PASS — Owner attested** | Exact remediation MSI; no account label or external ID retained. |
| Local WeCom disconnect | **PASS — Owner attested** | Exact remediation MSI; local disconnect only, with no remote-revocation claim. |
| Failure/retry | **Observed, not attributed to the attested final pass** | Local metadata contains cancelled QR attempts ending in stable `CHANNEL_AUTH_CANCELLED`; no raw QR or account value is stored. |

This matrix deliberately does not turn nearby durable state into unreported manual
steps.

## Sanitized post-smoke inspection

Read-only inspection covered the local YORVA SQLite file and the available YORVA
structured install log:

- the Owner reported no QR, Secret, pairing-code or account-information disclosure
  during the exact-MSI flows;
- SQLite integrity: `ok`;
- `channel_bindings.metadata_json` remained exactly `{}` for observed rows;
- no secret, QR or pairing-code column exists in the Phase 6 projection schema;
- four currently available Hermes-native Weixin token values were compared by exact
  value against both inspected YORVA surfaces: zero occurrences;
- generic QR-payload and channel-secret markers in SQLite: zero occurrences;
- no current WeCom Bot ID/Secret remained available for an exact-value comparison, so
  a WeCom exact-value zero-occurrence claim is not made;
- no searched value, derived low-entropy hash or raw provider/Hermes output was printed
  or committed.

The inspection supports the no-duplicate-secret design for the values still available;
it is not a substitute for a contemporaneous controlled-sentinel check of the missing
WeCom value.

## Remaining limitations

- the exact action clock was not supplied; 2026-08-24 is the supplemental attestation
  and test-record date;
- the tested remediation MSI is exact, while the displayed Desktop version was not
  separately transcribed;
- the exact Profile and default/named selection were not supplied;
- a real message-response check was not separately supplied;
- the Owner attested no disclosure during the exact-MSI test, while an independent
  exact-value WeCom comparison cannot be reconstructed after credential removal;
- this evidence must be combined with non-secret exact-candidate CI, package inspection,
  lifecycle/Desktop smoke and independent audit. It is not by itself a Phase Gate.
