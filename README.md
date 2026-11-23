# YouTube Music → Discord

[![CI](https://github.com/tobsatop/discord-activity/actions/workflows/ci.yml/badge.svg)](https://github.com/tobsatop/discord-activity/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**ภาษาไทย** | [English](README.en.md)

โชว์เพลงที่กำลังฟังบน YouTube Music (ใน Chrome) เป็นสถานะ **"Listening to …"** ใน Discord
ใช้ได้ทั้ง **Windows** และ **macOS**

<p><img src="docs/discord-status.png" alt="ตัวอย่างสถานะใน Discord: กำลังฟัง YouTube Music — ชื่อเพลง ศิลปิน อัลบั้ม และแถบเวลา"></p>

## ทำอะไรได้

- โชว์ชื่อเพลง ศิลปิน ปกอัลบั้ม แถบเวลา และปุ่ม "Listen on YouTube Music"
- หยุดเพลง ช่วงโฆษณา ปิดแท็บ หรือปิด Chrome → สถานะหายเอง
- ไม่ต้องเปิดโปรแกรมค้างไว้ Chrome เปิดโปรแกรมเสริมให้เองเมื่อเปิด YouTube Music
- ใช้ได้กับ **music.youtube.com** เท่านั้น (ไม่รองรับ youtube.com ธรรมดา)

## สิ่งที่ต้องมี

| อะไร | หมายเหตุ |
|---|---|
| Google Chrome บนคอม | ทดสอบแล้วบน macOS และ Windows |
| แอป Discord บนคอม | ต้องเป็นแอป ใช้ Discord บนเว็บไม่ได้ |
| Node.js **22.18 ขึ้นไป** | แนะนำ **24 LTS** จาก https://nodejs.org |
| Application ID ของ Discord | สร้างฟรีในขั้นที่ 1 |

## ติดตั้ง (ทำครั้งเดียวต่อเครื่อง)

### ขั้นที่ 1 — สร้าง Application ใน Discord (ทำครั้งเดียว ใช้ได้ทุกเครื่อง)

1. เข้า https://discord.com/developers/applications แล้ว login ด้วยบัญชี Discord
2. กด **New Application** → ตั้งชื่อ `YouTube Music` → ติ๊กยอมรับ → **Create**
   ชื่อนี้จะโชว์บนการ์ดสถานะ เช่น "Listening to YouTube Music"
3. หน้า **General Information** → กด **Copy** ที่ช่อง **Application ID** (ตัวเลขยาว 17–20 หลัก) เก็บไว้

ไม่ต้องตั้งค่าอย่างอื่น ไม่ต้องสร้าง bot ไม่ต้องอัปโหลดรูป

### ขั้นที่ 2 — ดาวน์โหลดโค้ด

เลือกวิธีใดวิธีหนึ่ง:

- **มี git:** `git clone https://github.com/tobsatop/discord-activity.git`
- **ไม่มี git:** หน้า GitHub → ปุ่ม **Code** → **Download ZIP** → **แตกไฟล์ ZIP ก่อน** (Windows: คลิกขวา → Extract All)

> วางโฟลเดอร์ไว้ที่ถาวรก่อนติดตั้ง ถ้าย้ายหรือเปลี่ยนชื่อโฟลเดอร์ทีหลัง ต้องทำขั้นที่ 3 และ 4 ใหม่

### ขั้นที่ 3 — ติดตั้งโปรแกรมเสริม

<details open>
<summary><b>Windows</b></summary>

1. ติดตั้ง Node.js 24 LTS จาก https://nodejs.org (ไฟล์ `.msi` กด Next ไปจนจบ ใช้ค่าเริ่มต้นได้เลย)
2. เปิดโฟลเดอร์โปรเจกต์ใน File Explorer → **คลิกขวาที่ที่ว่าง** → **Open in Terminal**
   (Windows 10: กด `Shift` ค้างแล้วคลิกขวา → **Open PowerShell window here**)
3. เช็คเวอร์ชัน Node:
   ```
   node -v
   ```
   ต้องได้ `v22.18` ขึ้นไป ถ้าขึ้นว่า `node is not recognized` ให้ปิดแล้วเปิด Terminal ใหม่ (หรือ restart เครื่อง)
4. รันตัวติดตั้ง:
   ```
   node scripts\install.ts
   ```
   **อย่าใช้ `npm run setup` ใน PowerShell** เพราะ PowerShell บล็อกคำสั่ง npm ไว้ (error: *running scripts is disabled on this system*)
   ถ้าใช้ Command Prompt (cmd) จะใช้ `npm run setup` ได้
5. วาง Application ID (คลิกขวา หรือ `Ctrl + V`) → Enter
6. ขึ้นข้อความ **"ลงทะเบียนโปรแกรมเสริมกับ Chrome, Edge แล้ว"** = สำเร็จ

ตัวติดตั้งเก็บตัวเปิดโปรแกรมไว้ที่ `%LOCALAPPDATA%\ytm-discord-presence\` และลงทะเบียนใน Registry ของผู้ใช้ (HKCU) ไม่ต้องใช้สิทธิ์ Administrator

</details>

<details open>
<summary><b>macOS</b></summary>

1. ติดตั้ง Node.js 24 LTS จาก https://nodejs.org (ไฟล์ `.pkg`) หรือใช้ `brew install node`
2. เปิด Chrome อย่างน้อยหนึ่งครั้งก่อน (ตัวติดตั้งจะหาโฟลเดอร์ของ Chrome)
3. เปิดแอป **Terminal** (กด `Cmd + Space` พิมพ์ Terminal)
4. พิมพ์ `cd ` (มีเว้นวรรคต่อท้าย) แล้ว **ลากโฟลเดอร์โปรเจกต์มาวาง** ในหน้าต่าง Terminal → กด Enter
5. เช็คเวอร์ชัน Node:
   ```
   node -v
   ```
   ต้องได้ `v22.18` ขึ้นไป
6. รันตัวติดตั้ง:
   ```
   npm run setup
   ```
7. วาง Application ID (`Cmd + V`) → Enter
8. ขึ้นข้อความ **"ลงทะเบียนโปรแกรมเสริมกับ Chrome … แล้ว"** = สำเร็จ

ตัวติดตั้งเก็บตัวเปิดโปรแกรมไว้ที่ `~/Library/Application Support/ytm-discord-presence/` จึงวางโปรเจกต์ไว้ใน Documents ได้
ถ้าใช้ nvm แล้วลบหรือเปลี่ยนเวอร์ชัน Node ให้รันตัวติดตั้งใหม่

</details>

### ขั้นที่ 4 — โหลดส่วนขยายใน Chrome

1. พิมพ์ `chrome://extensions` ในช่อง address
2. เปิดสวิตช์ **Developer mode** มุมขวาบน
3. กด **Load unpacked** → เลือกโฟลเดอร์ **`extension`** ที่อยู่ข้างในโปรเจกต์
   (อย่าเลือกโฟลเดอร์โปรเจกต์ทั้งก้อน จะขึ้น error ว่า *Manifest file is missing*)
4. กดรูปจิ๊กซอว์บนแถบ Chrome → กดหมุด 📌 ที่ **YouTube Music → Discord** เพื่อให้เห็นไอคอนสถานะ
5. ถ้าเปิดแท็บ YouTube Music ค้างไว้ก่อนแล้ว ให้ **refresh แท็บนั้น 1 ครั้ง**

### ขั้นที่ 5 — ลองใช้

1. เปิดแอป Discord และ login (Windows: **อย่าเปิด Discord แบบ Run as administrator**)
2. Discord → User Settings → **Activity Privacy** → เปิดการแชร์กิจกรรม
   (ชื่อตัวเลือกประมาณ *Share my activity* หรือ *Display current activity as a status message*)
3. เปิด https://music.youtube.com แล้วเล่นเพลง
4. ภายในไม่กี่วินาที ไอคอนส่วนขยายเปลี่ยนเป็น **สีแดง** และโปรไฟล์ Discord ขึ้น "Listening to ชื่อเพลง"

## ใช้งานทุกวัน

**ไม่ต้องรันตัวติดตั้งทุกครั้งที่เปิดเครื่อง** ทำขั้นที่ 1–5 ครั้งเดียวต่อเครื่องพอ

หลังเปิดเครื่องแค่:

1. เปิดแอป Discord
2. เปิด Chrome แล้วเล่นเพลงที่ music.youtube.com

Chrome จะเปิดโปรแกรมเสริมให้เองเมื่อมีแท็บ YouTube Music และปิดให้เองเมื่อปิดแท็บ

ตั้งให้ Discord เปิดเองตอนเปิดเครื่องได้:
- Windows: Discord → User Settings → Windows Settings → เปิด **Open Discord**
- macOS: System Settings → General → Login Items → เพิ่ม Discord

รันตัวติดตั้งใหม่ (ขั้นที่ 3) เฉพาะเมื่อ:
- ย้ายหรือเปลี่ยนชื่อโฟลเดอร์โปรเจกต์
- ลบหรือย้าย Node หรืออัปเดต Node ผ่าน nvm / Homebrew
  (อัปเดตด้วยไฟล์ติดตั้งจาก nodejs.org ไม่ต้องรันใหม่)
- เปลี่ยน Application ID (ดูหัวข้อ "เปลี่ยน Application ID")

## ไอคอนบอกอะไร

ชี้เมาส์ค้างที่ไอคอนเพื่อดูรายละเอียด

| ไอคอน | ความหมาย |
|---|---|
| <img src="extension/icons/icon-32.png" width="20" alt="แดง"> สีแดง | กำลังโชว์ใน Discord |
| <img src="extension/icons/idle-32.png" width="20" alt="เทา"> สีเทา | ยังไม่มีแท็บ YouTube Music หรือยังไม่ได้เล่นเพลง |
| <img src="extension/icons/warn-32.png" width="20" alt="จุดส้ม"> เทา + จุดส้ม | ไม่เจอแอป Discord บนเครื่อง |
| <img src="extension/icons/error-32.png" width="20" alt="จุดแดง"> เทา + จุดแดง | เปิดโปรแกรมเสริมไม่ได้ หรือ Discord ไม่รับ Application ID |

## แก้ปัญหา

| อาการ | วิธีแก้ |
|---|---|
| จุดแดง "เปิดโปรแกรมเสริมไม่ได้" | ยังไม่ได้ทำขั้นที่ 3 ย้ายโฟลเดอร์ หรือลบ Node ไป → ทำขั้นที่ 3 ใหม่ แล้วรอประมาณ 30 วินาที |
| จุดแดง "Discord ไม่รับการเชื่อมต่อ … Application ID" | Application ID ผิด → ดูหัวข้อ "เปลี่ยน Application ID" ด้านล่าง |
| จุดส้ม "ยังไม่เจอแอป Discord" | เปิดแอป Discord แล้วรอประมาณ 15 วินาที ถ้ายังไม่หาย: Windows ให้ปิด Discord ที่เปิดแบบ administrator แล้วเปิดใหม่แบบปกติ |
| ไอคอนเป็นสีเทาทั้งที่เล่นเพลงอยู่ | refresh แท็บ YouTube Music (จำเป็นทุกครั้งหลังโหลดหรือ reload ส่วนขยาย) |
| ไอคอนสีแดงแต่เพื่อนไม่เห็น | เปิด Activity Privacy ตามขั้นที่ 5 และอย่าตั้งสถานะเป็น Invisible |
| กดปุ่ม "Listen on YouTube Music" บนโปรไฟล์ตัวเองไม่ได้ | ปกติของ Discord คนอื่นกดได้ |
| สถานะเปลี่ยนช้าเมื่อกดข้ามเพลงรัว ๆ | Discord รับได้ 5 ครั้งต่อ 20 วินาที เดี๋ยวตามมาเอง |
| ตัวติดตั้งขึ้น `Unknown file extension ".ts"` | Node เก่าเกินไป → ติดตั้ง Node 24 LTS |
| ตัวติดตั้งขึ้น "ไม่พบ Chrome บนเครื่องนี้" (macOS) | เปิด Chrome หนึ่งครั้งแล้วรันใหม่ |
| ใช้ Chrome โปรไฟล์ที่ทำงานหรือโรงเรียนดูแล | องค์กรอาจปิด Developer mode หรือ native messaging → ลองโปรไฟล์ส่วนตัว |

ยังไม่หาย: ดู log ที่ `host/host.log` ในโฟลเดอร์โปรเจกต์

## เปลี่ยน Application ID

ตัวติดตั้งจะจำ ID เดิมไว้ ถ้าจะเปลี่ยนให้ใส่ ID ใหม่ต่อท้ายคำสั่ง:

- Windows PowerShell: `node scripts\install.ts 123456789012345678`
- macOS: `npm run setup -- 123456789012345678`

จากนั้นปิดแท็บ YouTube Music **ทุกแท็บ** แล้วเปิดใหม่ เพื่อให้โปรแกรมเสริมเริ่มใหม่ด้วย ID ใหม่

## อัปเดตเป็นเวอร์ชันใหม่

- ดูเวอร์ชันที่ใช้อยู่: `chrome://extensions` → เลขข้างชื่อส่วนขยาย
- ดูว่ามีอะไรใหม่: [CHANGELOG.md](CHANGELOG.md)
- อัปเดตแล้ว Application ID ไม่หาย และไม่ต้องติดตั้งใหม่

**ดาวน์โหลดแบบ ZIP (ไม่ใช้ git)** — ใช้คำสั่งเดียว:

1. เปิด Terminal ในโฟลเดอร์โปรเจกต์ (แบบเดียวกับขั้นที่ 3) แล้วรัน
   - Windows PowerShell: `node scripts\update.ts`
   - macOS: `npm run update`

   คำสั่งนี้โหลดเวอร์ชันล่าสุดจาก GitHub มาทับเฉพาะไฟล์ที่เปลี่ยน และลงทะเบียนโปรแกรมเสริมให้ใหม่
2. `chrome://extensions` → กดปุ่มหมุน ↻ บนการ์ดของส่วนขยาย
3. refresh แท็บ YouTube Music

**ใช้ git:** `git pull` แล้วทำข้อ 2–3

<details>
<summary>ถ้าใช้เวอร์ชัน 0.1.0 อยู่ (ยังไม่มี <code>scripts/update.ts</code>) — อัปเดตด้วยมือครั้งเดียว</summary>

1. หน้า GitHub → **Code** → **Download ZIP** → คลิกขวา → **Extract All**
2. เปิดเข้าไปจนเจอโฟลเดอร์ที่มี `extension`, `host`, `scripts` อยู่ข้างใน
3. เลือกทุกอย่างในนั้น → Copy → วางในโฟลเดอร์เดิม → เลือก **Replace the files in the destination**
4. ทำข้อ 2–3 ด้านบน

ระวัง: ถ้าแตก ZIP เป็นโฟลเดอร์ใหม่แล้วไม่ได้ copy ทับ Chrome จะยังใช้โฟลเดอร์เก่าอยู่
ครั้งต่อไปใช้ `node scripts\update.ts` ได้เลย

</details>

## ถอนการติดตั้ง

1. Windows PowerShell: `node scripts\install.ts --uninstall` · macOS: `npm run setup -- --uninstall`
2. `chrome://extensions` → **Remove** ส่วนขยาย
3. ลบโฟลเดอร์โปรเจกต์ และลบ Application ใน Discord Developer Portal ได้ถ้าไม่ใช้แล้ว

## ความเป็นส่วนตัว

- ส่วนขยายทำงานเฉพาะบน music.youtube.com และขอสิทธิ์แค่ `nativeMessaging` (คุยกับโปรแกรมเสริมบนเครื่อง)
- ข้อมูลเพลงส่งไปแค่แอป Discord บนเครื่องคุณ ไม่มีเซิร์ฟเวอร์อื่น Discord จะโชว์ให้เพื่อนเห็นตามการตั้งค่าของคุณ
- `host/host.log` เก็บชื่อเพลงล่าสุดไว้บนเครื่องเพื่อใช้แก้ปัญหา (เกิน 1 MB จะล้างเอง)

## ทำงานยังไง

```
music.youtube.com ─▶ ส่วนขยาย Chrome ─▶ โปรแกรมเสริม (Chrome เปิดให้เอง) ─▶ แอป Discord
```

| ไฟล์ | หน้าที่ |
|---|---|
| `extension/page-reader.js` | อ่านเพลง ศิลปิน ปก และเวลาที่เล่นจากหน้า YouTube Music |
| `extension/bridge.js` | ส่งต่อข้อมูลจากหน้าเว็บไปที่ background |
| `extension/background.js` | เลือกเพลงที่จะโชว์ ส่งให้โปรแกรมเสริม และอัปเดตไอคอน |
| `host/main.ts` | โปรแกรมเสริมที่ Chrome เปิดให้เอง (native messaging) |
| `host/discord-ipc.ts` | คุยกับแอป Discord ผ่าน IPC ต่อใหม่เองเมื่อหลุด และจำกัดความถี่ตามที่ Discord กำหนด |
| `host/activity.ts` | แปลงข้อมูลเพลงเป็นสถานะ Discord และตรวจข้อมูลที่ได้จากเบราว์เซอร์ |
| `extension/icons/` | ไอคอนต้นฉบับ `icon.svg` และ PNG ของแต่ละสถานะ |
| `scripts/install.ts` | ลงทะเบียนโปรแกรมเสริมกับ Chrome (Windows / macOS / Linux) |
| `scripts/update.ts` | อัปเดตโฟลเดอร์ที่โหลดแบบ ZIP เป็นเวอร์ชันล่าสุดจาก GitHub |
| `scripts/icons.ts` | สร้าง PNG ทุกสถานะจาก `icon.svg` (ต้องมี ImageMagick) |

Node รันไฟล์ `.ts` ได้ตรง ๆ จึงไม่ต้อง build ส่วนขยายเขียนเป็น `.js` แต่ TypeScript ตรวจ type ให้ด้วย

## สำหรับนักพัฒนา

```
npm install      # ติดตั้งแค่ TypeScript กับ type definitions
npm run check    # ตรวจ type ทั้งสองฝั่ง
npm test         # ทดสอบโปรแกรมเสริมกับ Discord จำลอง
```

- แก้ไฟล์ใน `extension/` → กด ↻ ที่ `chrome://extensions` แล้ว refresh แท็บ YouTube Music
- แก้ไฟล์ใน `host/` → ปิดแท็บ YouTube Music ทุกแท็บแล้วเปิดใหม่ (โปรแกรมเสริมจะเริ่มใหม่)
- ไอคอน: แก้ `extension/icons/icon.svg` แล้วรัน `npm run icons` (ต้องมี ImageMagick) จะได้ไอคอนทุกสถานะ (`icon-*` แดง, `idle-*` เทา, `warn-*` จุดส้ม, `error-*` จุดแดง) หรือวาง PNG ของตัวเองทับไฟล์เหล่านี้แล้วกด ↻
- Windows PowerShell: ใช้ `npm.cmd` แทน `npm` เช่น `npm.cmd test` (ปิดแอป Discord ก่อนรันเทสต์บน Windows เพราะ Discord จำลองใช้ pipe ชื่อเดียวกับของจริง)
- ทุกครั้งที่ push GitHub Actions จะรัน `npm run check` และ `npm test` บน Windows, macOS และ Linux ให้เอง
- ออกเวอร์ชันใหม่: เปลี่ยนเลขใน `package.json` และ `extension/manifest.json` ให้ตรงกัน เขียนรายการใน `CHANGELOG.md` และแก้ README ให้ตรงกันทั้งสองภาษา (`README.md` ไทย, `README.en.md` อังกฤษ)

## License

[MIT](LICENSE) — ใช้ แก้ และแจกต่อได้ โดยใส่เครดิต

YouTube Music เป็นเครื่องหมายการค้าของ Google LLC และ Discord เป็นเครื่องหมายการค้าของ Discord Inc.
โปรเจกต์นี้ไม่ได้เกี่ยวข้อง หรือได้รับการรับรองจากทั้งสองบริษัท
