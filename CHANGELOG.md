# Changelog

เวอร์ชันใหม่อยู่บนสุด ดูเวอร์ชันที่ใช้อยู่ได้ที่ `chrome://extensions` (เลขข้างชื่อส่วนขยาย)
วิธีอัปเดตอยู่ใน [README หัวข้ออัปเดต](README.md)

## 0.3.1 — 2026-11-24

### เปลี่ยน
- ชื่อไฟล์ตัวติดตั้งอ่านง่ายขึ้น
  - Windows: `YouTube-Music-Discord-Setup.exe` (เครื่อง ARM ใช้ `YouTube-Music-Discord-Setup-ARM.exe`)
  - macOS: `YouTube-Music-Discord-Setup-Mac.zip` แตกแล้วได้แอป **YouTube Music Discord Setup**
- macOS: ตัวติดตั้งเป็นแอป ใช้กล่องข้อความแทนหน้าต่าง Terminal
- ตัวติดตั้งมีไอคอนของตัวเอง (แดงชมพู โน้ตดนตรี ป้ายสถานะสีม่วง)
- Windows: ไฟล์ .exe มีข้อมูลโปรแกรม (ชื่อ คำอธิบาย รุ่น ผู้ทำ) ดูได้ที่คลิกขวา → Properties → Details

### อัปเดตจาก 0.3.0
- โหลดตัวติดตั้งใหม่แล้วดับเบิลคลิก จากนั้นกด ↻ บนการ์ดที่ `chrome://extensions` และ refresh แท็บ YouTube Music

## 0.3.0 — 2026-11-24

### เปลี่ยน
- ไม่ต้องติดตั้ง Node.js และไม่ต้องพิมพ์คำสั่งใน Terminal แล้ว
  - โหลดไฟล์เดียวจากหน้า Releases บน GitHub แล้วดับเบิลคลิก
  - Windows: `ytm-discord-windows-amd64.exe` (เครื่อง ARM ใช้ `ytm-discord-windows-arm64.exe`)
  - macOS: `ytm-discord-macos.zip` ใช้ได้ทั้ง Apple Silicon และ Intel
  - วาง Application ID แล้วโปรแกรมติดตั้งตัวเอง แตกส่วนขยายไว้ให้ และเปิดโฟลเดอร์ส่วนขยายให้
- โปรแกรมเสริมเขียนใหม่ด้วยภาษา Go (เดิมเป็น Node.js) ทำงานเหมือนเดิมทุกอย่าง
- ไฟล์ทั้งหมดย้ายไปอยู่ในโฟลเดอร์ข้อมูลแอปของระบบ
  - Windows: `%LOCALAPPDATA%\ytm-discord-presence\`
  - macOS: `~/Library/Application Support/ytm-discord-presence/`
  - ข้างในมี `config.json` (Application ID), `host.log` และโฟลเดอร์ `extension` ที่ใช้กด Load unpacked
- `host.log` ไม่เก็บชื่อผู้ใช้ Discord แล้ว เพราะคนจะแนบ log มาตอนแจ้งปัญหา

### เพิ่ม
- README ภาษาอังกฤษ (`README.en.md`) และลิงก์สลับภาษาด้านบนของทั้งสองไฟล์
- License แบบ MIT
- ตรวจโค้ดอัตโนมัติด้วย GitHub Actions บน Windows, macOS และ Linux
  - บน Windows ทดสอบตัวติดตั้งด้วย: ลงทะเบียน → ตรวจ → ถอน
- push tag เวอร์ชันแล้ว GitHub สร้างไฟล์ให้โหลดและหน้า Release เอง
- แบบฟอร์มแจ้งปัญหาบน GitHub (ไทย/อังกฤษ)

### ลบ
- `npm run setup` / `node scripts\install.ts` และคำสั่งอัปเดต `scripts/update.ts`
  - อัปเดตต่อจากนี้: โหลดไฟล์ใหม่แล้วดับเบิลคลิกอีกครั้ง
  - ถอนการติดตั้ง: รันโปรแกรมพร้อม `--uninstall`

### อัปเดตจาก 0.2.0
1. โหลดไฟล์ของเครื่องตัวเองจากหน้า Releases แล้วดับเบิลคลิก
   - ใช้ Application ID เดิมต่อให้เอง ไม่ต้องวางใหม่
2. ส่วนขยายย้ายไปอยู่ในโฟลเดอร์ใหม่ที่โปรแกรมบอก
   - ที่ `chrome://extensions` กด Remove การ์ดเดิม
   - กด Load unpacked แล้วเลือกโฟลเดอร์ใหม่
3. refresh แท็บ YouTube Music
4. ลบโฟลเดอร์โปรเจกต์เดิมได้เลย และถอน Node.js ได้ถ้าไม่ได้ใช้ทำอย่างอื่น

## 0.2.0 — 2026-11-24

### เพิ่ม
- ไอคอนส่วนขยายของตัวเอง: วงกลมแดง + โน้ตดนตรีสีขาว (วาดเอง ไม่ใช้โลโก้ YouTube Music)
- ไอคอนบนแถบ Chrome เปลี่ยนสีตามสถานะ แทนป้าย ON / ! ที่บังโลโก้
  - แดง = กำลังโชว์ใน Discord
  - เทา = ยังไม่ได้เล่นเพลง
  - เทา + จุดส้ม = ไม่เจอแอป Discord
  - เทา + จุดแดง = เปิดโปรแกรมเสริมไม่ได้ หรือ Discord ไม่รับ Application ID
- คำสั่งอัปเดตคำสั่งเดียว สำหรับคนที่โหลดแบบ ZIP: `node scripts\update.ts` (Windows) / `npm run update` (macOS)
  - โหลดเวอร์ชันล่าสุดจาก GitHub มาทับเฉพาะไฟล์ที่เปลี่ยน
  - Application ID ไม่หาย
  - ลงทะเบียนโปรแกรมเสริมให้ใหม่เอง
- `npm run icons` สร้างไอคอนทุกสถานะจาก `extension/icons/icon.svg`

### เปลี่ยน
- README เขียนใหม่ทั้งไฟล์
  - ขั้นตอนทีละขั้น Windows ก่อน macOS
  - มีหัวข้อ "ใช้งานทุกวัน" ตารางแก้ปัญหา และรูปตัวอย่าง
- ข้อความแนะนำบนไอคอนชี้ไปที่ README แทน `npm run setup` เพราะคำสั่งนี้ใช้ใน Windows PowerShell ไม่ได้

### อัปเดตจาก 0.1.0
- ZIP: เวอร์ชัน 0.1.0 ยังไม่มีคำสั่งอัปเดต ต้องโหลด ZIP มา copy ทับเองครั้งเดียว (README หัวข้ออัปเดต)
- git: `git pull`
- จากนั้นกด ↻ ที่ `chrome://extensions` แล้ว refresh แท็บ YouTube Music

## 0.1.0 — 2026-11-24

เวอร์ชันแรก

- ส่วนขยาย Chrome อ่านเพลงจาก music.youtube.com
- Discord โชว์ "Listening to ชื่อเพลง" พร้อมศิลปิน ปกอัลบั้ม แถบเวลา และปุ่ม "Listen on YouTube Music"
- หยุดเพลง ช่วงโฆษณา ปิดแท็บ หรือปิด Chrome → สถานะหายเอง
- โปรแกรมเสริม (native messaging) ที่ Chrome เปิดให้เอง
  - คุยกับแอป Discord ผ่าน IPC
  - ต่อใหม่เองเมื่อหลุด
  - จำกัดการส่งไว้ที่ 5 ครั้งต่อ 20 วินาทีตามที่ Discord กำหนด
- ตัวติดตั้ง `scripts/install.ts` สำหรับ Windows, macOS และ Linux

### แก้ก่อนปล่อย
- macOS: Chrome ไม่ยอมเปิดโปรแกรมที่อยู่ในโฟลเดอร์ Documents
  → ตัวติดตั้งเก็บตัวเปิดโปรแกรมไว้ในโฟลเดอร์ข้อมูลแอปของระบบแทน
- Windows: PowerShell บล็อกคำสั่ง `npm`
  → README บอกให้ใช้ `node scripts\install.ts`
