import { execFileSync } from "node:child_process";
import { mkdirSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = dirname(dirname(fileURLToPath(import.meta.url)));
const desktopRoot = join(repositoryRoot, "apps", "desktop");
const productVersion = JSON.parse(readFileSync(join(desktopRoot, "package.json"), "utf8")).version;
const tauriVersion = JSON.parse(readFileSync(join(desktopRoot, "src-tauri", "tauri.conf.json"), "utf8")).version;
const cargoVersion = readFileSync(join(desktopRoot, "src-tauri", "Cargo.toml"), "utf8")
  .match(/^version\s*=\s*"([^"]+)"/mu)?.[1];
if (typeof productVersion !== "string"
  || !/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/u.test(productVersion)
  || productVersion !== tauriVersion || productVersion !== cargoVersion) {
  throw new Error("Desktop, Tauri, Rust, and sidecar product versions must agree.");
}
const rustVersion = execFileSync("rustc", ["-vV"], { encoding: "utf8" });
const hostLine = rustVersion.split(/\r?\n/u).find((line) => line.startsWith("host: "));

if (!hostLine) {
  throw new Error("Could not determine the Rust host target.");
}

const target = hostLine.slice("host: ".length).trim();
if (!/^[a-zA-Z0-9_.-]+$/u.test(target)) {
  throw new Error(`Rust reported an invalid host target: ${target}`);
}

const extension = target.includes("windows") ? ".exe" : "";
const output = join(
  repositoryRoot,
  "apps",
  "desktop",
  "src-tauri",
  "binaries",
  `yorvad-${target}${extension}`,
);

mkdirSync(dirname(output), { recursive: true });
execFileSync("go", [
  "build", "-trimpath",
  "-ldflags", `-X github.com/YoLin02/yorva/services/node/internal/buildinfo.Version=${productVersion}`,
  "-o", output, "./cmd/yorvad",
], {
  cwd: join(repositoryRoot, "services", "node"),
  stdio: "inherit",
});
