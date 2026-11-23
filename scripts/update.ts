// Updates a ZIP-downloaded copy to the latest version on GitHub, then
// re-registers the helper. Keeps host/config.json (your Application ID) and logs.
//   Windows PowerShell: node scripts\update.ts      macOS: npm run update
// A folder made with `git clone` should use `git pull` instead.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const BRANCH = 'main';

/** "owner/repo" from the repository url in package.json. */
function repoSlug(): string {
  const pkg = JSON.parse(fs.readFileSync(path.join(ROOT, 'package.json'), 'utf8'));
  const url: string = typeof pkg.repository === 'string' ? pkg.repository : (pkg.repository?.url ?? '');
  const match = /github\.com[/:]([\w.-]+)\/([\w.-]+?)(?:\.git)?$/.exec(url);
  if (!match) throw new Error('package.json ไม่มีที่อยู่ GitHub (repository.url)');
  return `${match[1]}/${match[2]}`;
}

/** Copy every file under `from` into `to`, writing only files whose content differs. */
function syncInto(from: string, to: string, base = ''): string[] {
  const changed: string[] = [];
  for (const entry of fs.readdirSync(path.join(from, base), { withFileTypes: true })) {
    const rel = path.join(base, entry.name);
    if (entry.isDirectory()) {
      changed.push(...syncInto(from, to, rel));
      continue;
    }
    if (!entry.isFile()) continue;
    const data = fs.readFileSync(path.join(from, rel));
    const target = path.join(to, rel);
    if (fs.existsSync(target) && fs.readFileSync(target).equals(data)) continue;
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.writeFileSync(target, data);
    changed.push(rel);
  }
  return changed;
}

/** Download and unpack the latest version; returns the files that changed here. */
async function downloadLatest(slug: string): Promise<string[]> {
  const response = await fetch(`https://codeload.github.com/${slug}/tar.gz/refs/heads/${BRANCH}`);
  if (!response.ok) throw new Error(`ดาวน์โหลดไม่สำเร็จ (HTTP ${response.status})`);
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'ytmd-update-'));
  try {
    fs.writeFileSync(path.join(tmp, 'latest.tar.gz'), Buffer.from(await response.arrayBuffer()));
    // tar ships with Windows 10+, macOS and Linux. Relative path + cwd because
    // GNU tar (e.g. from Git for Windows) reads "C:\..." as a remote host.
    execFileSync('tar', ['-xzf', 'latest.tar.gz'], { cwd: tmp, stdio: 'ignore' });
    const top = fs.readdirSync(tmp, { withFileTypes: true }).find((entry) => entry.isDirectory());
    if (!top) throw new Error('ไฟล์ที่โหลดมาว่างเปล่า');
    return syncInto(path.join(tmp, top.name), ROOT);
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

async function main(): Promise<void> {
  if (fs.existsSync(path.join(ROOT, '.git'))) {
    console.log('โฟลเดอร์นี้ได้มาจาก git clone — อัปเดตด้วย git pull แทน');
    return;
  }

  const slug = repoSlug();
  console.log(`กำลังโหลดเวอร์ชันล่าสุดจาก github.com/${slug} …`);
  const changed = await downloadLatest(slug);
  if (changed.length === 0) {
    console.log('เป็นเวอร์ชันล่าสุดอยู่แล้ว ไม่ต้องทำอะไร');
    return;
  }
  console.log(`อัปเดตแล้ว ${changed.length} ไฟล์:`);
  for (const file of changed) console.log(`  ${file}`);

  console.log('');
  execFileSync(process.execPath, [path.join(ROOT, 'scripts', 'install.ts'), '--update'], { stdio: 'inherit' });

  console.log('');
  console.log('ขั้นต่อไป:');
  console.log('1. เปิด chrome://extensions แล้วกด ↻ บนการ์ด YouTube Music → Discord');
  console.log('2. refresh แท็บ YouTube Music');
}

main().catch((error: Error & { code?: string }) => {
  const reason = error.code === 'ENOENT' ? 'ไม่พบคำสั่ง tar บนเครื่อง' : error.message;
  console.error(`อัปเดตไม่สำเร็จ: ${reason}`);
  console.error('อัปเดตด้วยมือได้: โหลด ZIP จาก GitHub แล้ว copy ทับโฟลเดอร์นี้ (ดู README หัวข้ออัปเดต)');
  process.exit(1);
});
