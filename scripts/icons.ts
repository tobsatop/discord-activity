// Re-creates the extension's PNG icons from extension/icons/icon.svg.
// Usage: npm run icons   (needs ImageMagick: `magick` on PATH)
//
// icon-*.png   red: the extension's icon, and the toolbar icon while a song shows in Discord
// idle-*.png   grey: nothing showing right now
// warn-*.png   grey + amber dot: the Discord app isn't running
// error-*.png  grey + red dot: the helper can't start, or Discord refused it
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const ICONS = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', 'extension', 'icons');
const BASE = fs.readFileSync(path.join(ICONS, 'icon.svg'), 'utf8');
const RED = '#FF0033';
const GREY = '#80868B';
// A status dot in the corner, kept clear of the notes so nothing overlaps them.
const dot = (color: string) => `<circle cx="106" cy="106" r="18" fill="${color}" stroke="#FFFFFF" stroke-width="5"/>`;

const variants: Array<{ name: string; sizes: number[]; svg: string }> = [
  { name: 'icon', sizes: [16, 32, 48, 128], svg: BASE },
  { name: 'idle', sizes: [16, 32], svg: BASE.replace(RED, GREY) },
  { name: 'warn', sizes: [16, 32], svg: BASE.replace(RED, GREY).replace('</svg>', `${dot('#F9AB00')}</svg>`) },
  { name: 'error', sizes: [16, 32], svg: BASE.replace(RED, GREY).replace('</svg>', `${dot('#EA4335')}</svg>`) },
];

const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'ytmd-icons-'));
try {
  for (const variant of variants) {
    const svgFile = path.join(tmp, `${variant.name}.svg`);
    fs.writeFileSync(svgFile, variant.svg);
    for (const size of variant.sizes) {
      const out = path.join(ICONS, `${variant.name}-${size}.png`);
      execFileSync('magick', [
        '-background', 'none',
        '-density', '576', // render the 128-unit SVG at 1024 px, then scale down for clean edges
        svgFile,
        '-filter', 'Lanczos',
        '-resize', `${size}x${size}`,
        '-strip',
        `PNG32:${out}`,
      ]);
      console.log(`wrote ${path.relative(process.cwd(), out)}`);
    }
  }
} finally {
  fs.rmSync(tmp, { recursive: true, force: true });
}
