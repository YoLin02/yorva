import { createHash, generateKeyPairSync, verify } from "node:crypto";
import { mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { spawnSync } from "node:child_process";

const root = mkdtempSync(join(tmpdir(), "yorva-update-metadata-"));
const packagePath = join(root, "YORVA_0.4.0_x64_en-US.msi");
const artifactPath = join(root, "artifact.json");
const notesPath = join(root, "notes.txt");
const keyPath = join(root, "key.pem");
const outputPath = join(root, "update.json");
const bytes = Buffer.from("fixed-msi-fixture");
const { privateKey, publicKey } = generateKeyPairSync("ed25519");
const hash = createHash("sha256").update(bytes).digest("hex").toUpperCase();

writeFileSync(packagePath, bytes);
writeFileSync(notesPath, "Verified update fixture.");
writeFileSync(keyPath, privateKey.export({ format: "pem", type: "pkcs8" }));
writeFileSync(artifactPath, JSON.stringify({
  schemaVersion: 1,
  productVersion: "0.4.0",
  sourceCommit: "a".repeat(40),
  workingTreeDirty: false,
  fileName: "YORVA_0.4.0_x64_en-US.msi",
  sizeBytes: bytes.length,
  sha256: hash,
  signatureStatus: "NotSigned",
}));

const generator = resolve("scripts/create-yorva-update-metadata.mjs");
const result = spawnSync(process.execPath, [generator,
  "--artifact", artifactPath,
  "--release-notes", notesPath,
  "--private-key", keyPath,
  "--classification", "internal",
  "--output", outputPath,
], { encoding: "utf8" });
if (result.status !== 0) throw new Error(result.stderr || result.stdout);

const metadata = JSON.parse(readFileSync(outputPath, "utf8"));
const { signature, ...signed } = metadata;
if (!verify(null, Buffer.from(JSON.stringify(signed)), publicKey, Buffer.from(signature, "base64"))) {
  throw new Error("generated metadata signature did not verify");
}
if (metadata.package.url !== "https://github.com/YoLin02/yorva/releases/download/v0.4.0/YORVA_0.4.0_x64_en-US.msi") {
  throw new Error("generated metadata did not use the fixed release URL");
}
console.log("PASS signed fixed-source internal update metadata");

const qualificationOutput = join(root, "qualification.json");
const qualification = spawnSync(process.execPath, [generator,
  "--artifact", artifactPath,
  "--release-notes", notesPath,
  "--private-key", keyPath,
  "--classification", "internal",
  "--qualification", "windows-vm",
  "--output", qualificationOutput,
], { encoding: "utf8" });
if (qualification.status !== 0) throw new Error(qualification.stderr || qualification.stdout);
const qualificationMetadata = JSON.parse(readFileSync(qualificationOutput, "utf8"));
if (qualificationMetadata.package.url !== "https://10.0.2.2:18443/YORVA_0.4.0_x64_en-US.msi") {
  throw new Error("qualification metadata did not use the fixed disposable-VM URL");
}
console.log("PASS fixed disposable-VM qualification metadata");

const rejected = spawnSync(process.execPath, [generator,
  "--artifact", artifactPath,
  "--release-notes", notesPath,
  "--private-key", keyPath,
  "--output", join(root, "rejected.json"),
], { encoding: "utf8" });
if (rejected.status === 0 || !rejected.stderr.includes("explicit internal classification")) {
  throw new Error("unsigned artifact was not rejected without internal classification");
}
console.log("PASS unsigned public metadata rejection");
