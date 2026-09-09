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

// SPA mode (tanstackStart.spa.enabled): the build prerenders _shell.html with
// all window.$_TSR boot data. Serving it as index.html is the whole story.
const shellPath = path.join(clientDir, "_shell.html");
if (fs.existsSync(shellPath)) {
  let html = fs.readFileSync(shellPath, "utf8");
  const inlineScripts = [];
  const scriptRegex = /<script(?![^>]*\bsrc=)[^>]*>([\s\S]*?)<\/script>/gi;
  let match;
  let idx = 0;
  while ((match = scriptRegex.exec(html)) !== null) {
    const fullTag = match[0];
    const content = match[1].trim();
    if (!content) continue;
    const assetName = idx === 0 ? "tsr-scroll-restoration.js" : `tsr-inline-${idx}.js`;
    const assetPath = path.join(assetsDir, assetName);
    fs.writeFileSync(assetPath, content + "\n");
    const attrs = fullTag.match(/^<script([^>]*)>/)[1];
    const replacement = `<script${attrs} src="/assets/${assetName}"></script>`;
    html = html.replace(fullTag, replacement);
    inlineScripts.push(assetName);
    idx++;
  }
  fs.writeFileSync(path.join(clientDir, "index.html"), html);
  console.log(`SPA shell → index.html (externalized ${inlineScripts.length} inline scripts: ${inlineScripts.join(", ")})`);
  process.exit(0);
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
