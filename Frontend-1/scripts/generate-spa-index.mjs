/**
 * TanStack Start emits dist/client assets without a root index.html.
 * This generates a minimal SPA shell for nginx static hosting.
 *
 * Paths are resolved relative to Frontend-1/ (package root), not process.cwd(),
 * so the script works from repo root or from inside Frontend-1.
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const packageRoot = path.resolve(__dirname, "..");
const clientDir = path.join(packageRoot, "dist", "client");
const assetsDir = path.join(clientDir, "assets");

if (!fs.existsSync(assetsDir)) {
  console.error(`Missing ${assetsDir} — run \`npm run build\` first`);
  process.exit(1);
}

const files = fs.readdirSync(assetsDir);
const js = files.find((f) => /^index-.*\.js$/.test(f));
const cssIndex = files.find((f) => /^index-.*\.css$/.test(f));
const cssStyles = files.find((f) => /^styles-.*\.css$/.test(f));

if (!js) {
  console.error("No client entry index-*.js found in dist/client/assets");
  process.exit(1);
}

const links = [cssStyles, cssIndex]
  .filter(Boolean)
  .map((f) => `  <link rel="stylesheet" href="/assets/${f}" />`)
  .join("\n");

const html = `<!doctype html>
<html lang="sw">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
  <meta name="theme-color" content="#ffffff" />
  <title>Kikundi Bora</title>
  <meta name="description" content="Mfumo wa kidijitali wa kusimamia vikundi vya akiba na mikopo." />
  <link rel="manifest" href="/manifest.webmanifest" />
  <link rel="icon" type="image/png" href="/icon.png" />
  <link rel="apple-touch-icon" href="/icon.png" />
${links}
</head>
<body>
  <!-- TanStack Start client hydrates the document (hydrateRoot), not a #root div -->
  <script type="module" src="/assets/${js}"></script>
</body>
</html>
`;

const out = path.join(clientDir, "index.html");
fs.writeFileSync(out, html);
console.log(`SPA index.html generated → ${out}`);
console.log(`  entry: /assets/${js}`);
