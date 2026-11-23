// Re-creates the extension's PNG icons from extension/icons/icon.svg.
// Usage: npm run icons   (needs ImageMagick: `magick` on PATH)
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const ICONS = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', 'extension', 'icons');
const SIZES = [16, 32, 48, 128];

for (const size of SIZES) {
  const out = path.join(ICONS, `icon-${size}.png`);
  execFileSync('magick', [
    '-background', 'none',
    '-density', '576', // render the 128-unit SVG at 1024 px, then scale down for clean edges
    path.join(ICONS, 'icon.svg'),
    '-filter', 'Lanczos',
    '-resize', `${size}x${size}`,
    '-strip',
    `PNG32:${out}`,
  ]);
  console.log(`wrote ${path.relative(process.cwd(), out)}`);
}
