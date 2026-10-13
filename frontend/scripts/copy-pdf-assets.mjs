import { cp, mkdir } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import path from "node:path";
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
for (const dir of ["cmaps", "standard_fonts", "wasm", "iccs"]) {
  await mkdir(path.join(root, "public", "pdfjs"), { recursive: true });
  await cp(
    path.join(root, "node_modules", "pdfjs-dist", dir),
    path.join(root, "public", "pdfjs", dir),
    { recursive: true },
  );
}
await cp(
  path.join(root, "node_modules", "pdfjs-dist", "LICENSE"),
  path.join(root, "public", "licenses", "pdfjs-Apache-2.0.txt"),
);
