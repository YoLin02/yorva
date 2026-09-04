import { createHash, createPrivateKey, createPublicKey, sign } from "node:crypto";
import { readFileSync, statSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";

function fail(message) {
  throw new Error(message);
}

function options(argv) {
  const parsed = new Map();
  for (let index = 0; index < argv.length; index += 2) {
    const name = argv[index];
    const value = argv[index + 1];
    if (!name?.startsWith("--") || value === undefined) fail("arguments must be --name value pairs");
    if (parsed.has(name)) fail(`duplicate argument ${name}`);
    parsed.set(name, value);
  }
  return parsed;
}

const args = options(process.argv.slice(2));
const allowedArguments = new Set([
  "--artifact", "--release-notes", "--private-key", "--output", "--classification",
  "--signer-sha256", "--minimum-version", "--qualification",
]);
for (const name of args.keys()) {
  if (!allowedArguments.has(name)) fail(`unknown argument ${name}`);
}
const artifactPath = resolve(args.get("--artifact") ?? fail("--artifact is required"));
const notesPath = resolve(args.get("--release-notes") ?? fail("--release-notes is required"));
const keyPath = resolve(args.get("--private-key") ?? fail("--private-key is required"));
const outputPath = resolve(args.get("--output") ?? join(dirname(artifactPath), "yorva-update.json"));
const internal = args.get("--classification") === "internal";
const signerSha256 = args.get("--signer-sha256")?.toUpperCase();
const qualification = args.get("--qualification") === "windows-vm";
if (args.has("--classification") && !internal) fail("classification must be internal");
if (args.has("--qualification") && !qualification) fail("qualification must be windows-vm");

const artifact = JSON.parse(readFileSync(artifactPath, "utf8"));
const qualificationFixture = artifact.qualificationFixture === true;
if (artifact.schemaVersion !== 1 || artifact.qualificationBuild === true
    || (!qualificationFixture && artifact.workingTreeDirty !== false)
    || (qualificationFixture && !qualification)) {
  fail("update metadata requires a clean-source package artifact");
}
if (!/^\d+\.\d+\.\d+$/.test(artifact.productVersion)) fail("invalid artifact product version");
const expectedName = `YORVA_${artifact.productVersion}_x64_en-US.msi`;
if (artifact.fileName !== expectedName) fail("artifact filename does not match its product version");
if (!Number.isSafeInteger(artifact.sizeBytes) || artifact.sizeBytes <= 0 || artifact.sizeBytes > 512 * 1024 * 1024) {
  fail("artifact size is outside the updater bound");
}
if (!/^[A-Fa-f0-9]{64}$/.test(artifact.sha256)) fail("invalid artifact SHA-256");

const packagePath = join(dirname(artifactPath), artifact.fileName);
const packageInfo = statSync(packagePath);
const packageBytes = readFileSync(packagePath);
const actualSha256 = createHash("sha256").update(packageBytes).digest("hex").toUpperCase();
if (packageInfo.size !== artifact.sizeBytes || actualSha256 !== artifact.sha256.toUpperCase()) {
  fail("artifact manifest does not match the packaged MSI");
}

let authenticodePolicy;
if (artifact.signatureStatus === "Valid") {
  if (!/^[A-F0-9]{64}$/.test(signerSha256 ?? "")) fail("a valid signer SHA-256 is required");
  authenticodePolicy = "REQUIRED";
} else if (artifact.signatureStatus === "NotSigned" && internal) {
  authenticodePolicy = "INTERNAL_UNSIGNED";
} else {
  fail("unsigned packages require explicit internal classification");
}

const releaseNotes = readFileSync(notesPath, "utf8").trim();
if (releaseNotes.length === 0 || Buffer.byteLength(releaseNotes) > 8 * 1024) {
  fail("release notes must contain 1-8192 UTF-8 bytes");
}
const minimumVersion = args.get("--minimum-version") ?? "0.3.2";
if (!/^\d+\.\d+\.\d+$/.test(minimumVersion)) fail("invalid minimum version");

const packageRecord = {
  url: qualification
    ? `https://10.0.2.2:8443/${artifact.fileName}`
    : `https://github.com/YoLin02/yorva/releases/download/v${artifact.productVersion}/${artifact.fileName}`,
  sizeBytes: artifact.sizeBytes,
  sha256: artifact.sha256.toUpperCase(),
  authenticodePolicy,
  ...(authenticodePolicy === "REQUIRED" ? { signerSha256 } : {}),
};
const signed = {
  schemaVersion: 1,
  channel: "stable",
  version: artifact.productVersion,
  minimumVersion,
  publishedAtUtc: new Date().toISOString(),
  releaseNotes,
  package: packageRecord,
};

const privateKey = createPrivateKey(readFileSync(keyPath));
if (privateKey.asymmetricKeyType !== "ed25519") fail("the update key must be Ed25519");
const payload = Buffer.from(JSON.stringify(signed));
const signature = sign(null, payload, privateKey).toString("base64");
const publicDer = createPublicKey(privateKey).export({ format: "der", type: "spki" });
const ed25519Prefix = Buffer.from("302a300506032b6570032100", "hex");
if (publicDer.length !== ed25519Prefix.length + 32 || !publicDer.subarray(0, ed25519Prefix.length).equals(ed25519Prefix)) {
  fail("unexpected Ed25519 public key encoding");
}
const publicKey = publicDer.subarray(ed25519Prefix.length).toString("base64");

writeFileSync(outputPath, `${JSON.stringify({ ...signed, signature }, null, 2)}\n`, { encoding: "utf8", flag: "wx" });
process.stdout.write(`metadata ${outputPath}\nYORVA_UPDATE_PUBLIC_KEY=${publicKey}\n`);
