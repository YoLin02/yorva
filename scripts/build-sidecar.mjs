import { execFileSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = dirname(dirname(fileURLToPath(import.meta.url)));
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
execFileSync("go", ["build", "-trimpath", "-o", output, "./cmd/yorvad"], {
  cwd: join(repositoryRoot, "services", "node"),
  stdio: "inherit",
});
