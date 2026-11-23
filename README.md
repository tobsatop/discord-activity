# YouTube Music → Discord

โชว์เพลงที่กำลังฟังบน YouTube Music (ใน Chrome) เป็นสถานะ **"Listening to …"** ใน Discord
พร้อมปกอัลบั้ม แถบเวลา และปุ่ม "Listen on YouTube Music"

```
music.youtube.com ─▶ ส่วนขยาย Chrome ─▶ โปรแกรมเสริม (Chrome เปิดให้เอง) ─▶ แอป Discord
```

- หยุดเพลง → สถานะหาย / ปิดแท็บหรือปิด Chrome → สถานะหาย
- ไม่ต้องเปิดโปรแกรมอะไรค้างไว้เอง

## ติดตั้ง (ทำครั้งเดียวต่อเครื่อง)

1. ติดตั้ง **Node.js 24 LTS** (ใช้ 22.18 ขึ้นไปได้) และ **แอป Discord** (ใช้ Discord บนเว็บไม่ได้)
2. สร้างแอปใน Discord
   - เข้า https://discord.com/developers/applications → **New Application**
   - ตั้งชื่อ `YouTube Music` (ชื่อนี้จะโชว์ในโปรไฟล์)
   - คัดลอก **Application ID** จากหน้า General Information
3. เปิด Terminal ในโฟลเดอร์นี้ แล้วรัน
   ```
   npm run setup
   ```
   วาง Application ID เมื่อถูกถาม
   **Windows PowerShell:** ใช้ `node scripts\install.ts` แทน เพราะ PowerShell บล็อกคำสั่ง `npm` ไว้โดยค่าเริ่มต้น (ขึ้น error ว่า running scripts is disabled)
4. เปิด `chrome://extensions` → เปิด **Developer mode** → **Load unpacked** → เลือกโฟลเดอร์ `extension`
5. เปิดแอป Discord แล้วเล่นเพลงที่ https://music.youtube.com

**ย้ายไป Windows:** copy โฟลเดอร์นี้ไปทั้งโฟลเดอร์ (ไม่ต้องเอา `node_modules`) แล้วทำข้อ 1, 3, 4, 5 บน Windows
ใช้ Application ID เดิมได้ ถ้าย้ายโฟลเดอร์หรือเปลี่ยนเวอร์ชัน Node ให้รัน `npm run setup` ใหม่

## ถ้าไม่ขึ้น

- ดูไอคอนส่วนขยายบนแถบ Chrome: **ON** = ปกติ, **!** = มีปัญหา (ชี้เมาส์ค้างไว้เพื่อดูสาเหตุ)
- Discord → User Settings → **Activity Privacy** → เปิด "Share your detected activities with others"
- ดู log ได้ที่ `host/host.log`
- คนอื่นกดปุ่ม "Listen on YouTube Music" ได้ แต่ตัวเราเองอาจกดปุ่มบนโปรไฟล์ตัวเองไม่ได้ (Discord เป็นแบบนี้เอง)
- ถอนการติดตั้ง: `npm run setup -- --uninstall` (Windows PowerShell: `node scripts\install.ts --uninstall`) แล้วลบส่วนขยายใน `chrome://extensions`

## โครงสร้าง

| ไฟล์ | หน้าที่ |
|---|---|
| `extension/page-reader.js` | อ่านเพลง ศิลปิน ปก และเวลาที่เล่นจากหน้า YouTube Music |
| `extension/bridge.js` | ส่งต่อข้อมูลจากหน้าเว็บไปที่ background |
| `extension/background.js` | เลือกเพลงที่จะโชว์ ส่งให้โปรแกรมเสริม และอัปเดตไอคอน |
| `host/main.ts` | โปรแกรมเสริมที่ Chrome เปิดให้เอง (native messaging) |
| `host/discord-ipc.ts` | คุยกับแอป Discord ผ่าน IPC พร้อมต่อใหม่เองและจำกัดความถี่ (5 ครั้ง/20 วินาที) |
| `host/activity.ts` | แปลงข้อมูลเพลงเป็นสถานะ Discord และตรวจข้อมูลที่ได้จากเบราว์เซอร์ |
| `scripts/install.ts` | ลงทะเบียนโปรแกรมเสริมกับ Chrome (macOS / Windows / Linux) ตัวเปิดโปรแกรมเก็บไว้ในโฟลเดอร์ข้อมูลแอปของระบบ เพราะ macOS ไม่ให้ Chrome เปิดโปรแกรมใน Documents |

Node รันไฟล์ `.ts` ได้ตรง ๆ จึงไม่ต้อง build ส่วนขยายเขียนเป็น `.js` แต่ TypeScript ตรวจ type ให้ด้วย

## สำหรับพัฒนา

```
npm install      # ติดตั้งแค่ TypeScript กับ type definitions
npm run check    # ตรวจ type ทั้งสองฝั่ง
npm test         # ทดสอบโปรแกรมเสริมกับ Discord จำลอง
```

แก้ไฟล์ในโฟลเดอร์ `extension` แล้วกด reload ส่วนขยายใน `chrome://extensions` และ refresh แท็บ YouTube Music
