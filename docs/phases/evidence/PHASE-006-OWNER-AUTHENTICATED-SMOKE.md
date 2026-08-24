# Phase 6 Owner-Authenticated Weixin and WeCom Smoke

- Attestation received: 2026-08-24 (Asia/Shanghai)
- Evidence classification: **OWNER-ATTESTED PRODUCT-LEVEL EVIDENCE**
- Exact-candidate classification: **NOT ESTABLISHED**
- Redaction status: **SANITIZED; NO QR, SECRET, PAIRING CODE OR ACCOUNT VALUE**

## Owner attestation

The Repository Owner confirmed that validation with real Weixin and real WeCom
credentials/accounts passed. This record preserves that statement as Owner-provided
input. It does not add unreported test steps, account identity, message content or build
identity.

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

The Owner did not identify a tested MSI filename/hash or an exact commit/build ID. The
durable node row identifies a development build (`0.0.0-dev`) only. Therefore the real-
account result is not claimed as exact-candidate proof.

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
| Real Weixin validation | **PASS — Owner attested** | Product-level only; exact build and Profile not supplied. |
| Weixin QR generated/scanned/confirmed | **Not separately attested** | A succeeded `channel.connect` and connected Weixin safe projection exist locally, but ephemeral QR steps cannot be reconstructed and are not inferred. |
| Real message received an AI response | **Not evidenced** | Owner statement did not include a message-response check. |
| Sender pairing request/Desktop approval | **Not evidenced as a real-account step** | Batch 8A deterministic coverage is recorded in the implementation handoff. |
| Local Weixin disconnect | **Not evidenced** | No durable `channel.disconnect` Operation was found; some safe Weixin projections remain `CONNECTED`/`UNKNOWN`. |
| Real WeCom typed Bot ID/Secret validation | **PASS — Owner attested** | Product-level only; no credential value, exact build, Profile or step transcript retained. |
| WeCom reached `CONNECTED` | **Not independently reconstructable** | Current safe WeCom projections are `NOT_CONFIGURED`; the durable Operation schema does not record Channel type. |
| Local WeCom disconnect | **Not evidenced** | Current absence does not prove that the approved disconnect flow was exercised. |
| Failure/retry | **Observed, not attributed to the attested final pass** | Local metadata contains cancelled QR attempts ending in stable `CHANNEL_AUTH_CANCELLED`; no raw QR or account value is stored. |

This matrix deliberately does not turn nearby durable state into unreported manual
steps.

## Sanitized post-smoke inspection

Read-only inspection covered the local YORVA SQLite file and the available YORVA
structured install log:

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

- actual account-test date was not supplied; 2026-08-24 is the attestation date;
- displayed Desktop version, exact commit/build ID and tested MSI were not supplied;
- the exact Profile and default/named selection were not supplied;
- real message-response, pairing approval and local disconnect steps were not supplied;
- exact WeCom redaction comparison cannot be reconstructed after credential removal;
- this evidence must be combined with non-secret exact-candidate CI, package inspection,
  lifecycle/Desktop smoke and independent audit. It is not by itself a Phase Gate.
