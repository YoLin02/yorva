import { createServer } from "node:https";
import { readFileSync } from "node:fs";
import { basename, resolve } from "node:path";

function parse(argv) {
  const values = new Map();
  for (let index = 0; index < argv.length; index += 2) {
    if (!argv[index]?.startsWith("--") || argv[index + 1] === undefined) {
      throw new Error("arguments must be --name value pairs");
    }
    values.set(argv[index], argv[index + 1]);
  }
  return values;
}

const args = parse(process.argv.slice(2));
const allowed = new Set(["--metadata", "--package", "--certificate", "--key", "--mode"]);
for (const name of args.keys()) {
  if (!allowed.has(name)) throw new Error(`unknown argument ${name}`);
}
const required = (name) => resolve(args.get(name) ?? (() => { throw new Error(`${name} is required`); })());
const metadata = readFileSync(required("--metadata"));
const packagePath = required("--package");
const packageBytes = readFileSync(packagePath);
const packageName = basename(packagePath);
const mode = args.get("--mode") ?? "complete";
if (!new Set(["complete", "tampered", "interrupted"]).has(mode)) throw new Error("invalid mode");

const server = createServer({
  cert: readFileSync(required("--certificate")),
  key: readFileSync(required("--key")),
}, (request, response) => {
  if (request.method !== "GET") {
    response.writeHead(405).end();
    return;
  }
  if (request.url === "/yorva-update.json") {
    response.writeHead(200, { "Content-Type": "application/json", "Content-Length": metadata.length });
    response.end(metadata);
    return;
  }
  if (request.url !== `/${packageName}`) {
    response.writeHead(404).end();
    return;
  }
  response.writeHead(200, {
    "Content-Type": "application/octet-stream",
    "Content-Length": packageBytes.length,
  });
  if (mode === "tampered") {
    const changed = Buffer.from(packageBytes);
    changed[changed.length - 1] ^= 0xff;
    response.end(changed);
  } else if (mode === "interrupted") {
    response.write(packageBytes.subarray(0, Math.max(1, Math.floor(packageBytes.length / 4))));
    response.socket?.destroy();
  } else {
    response.end(packageBytes);
  }
});

server.listen(8443, "0.0.0.0", () => {
  process.stdout.write(`YORVA_UPDATE_FIXTURE_READY mode=${mode} package=${packageName}\n`);
});

for (const signal of ["SIGINT", "SIGTERM"]) {
  process.on(signal, () => server.close(() => process.exit(0)));
}
