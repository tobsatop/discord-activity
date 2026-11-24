package installer

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
)

// A made-up key; ExtensionID just hashes whatever the manifest holds.
const testKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAtest"

func testExtension() fstest.MapFS {
	return fstest.MapFS{
		"manifest.json":     {Data: []byte(`{"name":"test","key":"` + testKey + `"}`)},
		"background.js":     {Data: []byte("// bg")},
		"icons/icon-16.png": {Data: []byte("png")},
	}
}

type env struct {
	t        *testing.T
	home     string
	dataDir  string
	goos     string
	exe      string
	stdout   bytes.Buffer
	stderr   bytes.Buffer
	opened   []string
	extraEnv map[string]string
}

// newEnv lays out a fake home with the given browsers' profile folders.
// GOOS is "darwin" or "linux": the layouts differ only in folder names, so
// both run on any Unix test machine.
func newEnv(t *testing.T, goos string, browsers ...string) *env {
	if runtime.GOOS == "windows" {
		t.Skip("file-based browser registration is macOS/Linux; see registry_windows_test.go")
	}
	tmp := t.TempDir()
	e := &env{t: t, home: filepath.Join(tmp, "home"), dataDir: filepath.Join(tmp, "data"), goos: goos}
	for _, tg := range targets(goos, e.home, func(string) string { return "" }) {
		for _, b := range browsers {
			if tg.Browser == b {
				os.MkdirAll(filepath.Dir(filepath.Dir(tg.File)), 0o755)
			}
		}
	}
	e.exe = filepath.Join(tmp, "downloaded-program")
	os.WriteFile(e.exe, []byte("program v1"), 0o755)
	return e
}

func (e *env) getenv(k string) string {
	switch k {
	case EnvDataDir:
		return e.dataDir
	}
	return e.extraEnv[k]
}

func (e *env) run(stdin string, args ...string) int {
	e.stdout.Reset()
	e.stderr.Reset()
	return Run(Options{
		Args:        args,
		Stdin:       strings.NewReader(stdin),
		Stdout:      &e.stdout,
		Stderr:      &e.stderr,
		Getenv:      e.getenv,
		GOOS:        e.goos,
		Home:        e.home,
		Executable:  e.exe,
		Extension:   testExtension(),
		Interactive: len(args) == 0,
		Open:        func(dir string) { e.opened = append(e.opened, dir) },
	})
}

func (e *env) manifestFile(browser string) string {
	for _, tg := range targets(e.goos, e.home, func(string) string { return "" }) {
		if tg.Browser == browser {
			return tg.File
		}
	}
	e.t.Fatalf("no browser %s", browser)
	return ""
}

func (e *env) helperPath() string { return filepath.Join(e.dataDir, HelperName(e.goos)) }

func TestInstallWritesManifestsForInstalledBrowsers(t *testing.T) {
	for _, goos := range []string{"darwin", "linux"} {
		t.Run(goos, func(t *testing.T) {
			e := newEnv(t, goos, "Chrome", "Brave")
			if code := e.run("", "123456789012345678"); code != 0 {
				t.Fatalf("exit %d\n%s%s", code, e.stdout.String(), e.stderr.String())
			}
			id, _ := ExtensionID(testExtension()["manifest.json"].Data)
			for _, b := range []string{"Chrome", "Brave"} {
				data, err := os.ReadFile(e.manifestFile(b))
				if err != nil {
					t.Fatal(err)
				}
				var m map[string]any
				if err := json.Unmarshal(data, &m); err != nil {
					t.Fatal(err)
				}
				want := map[string]any{
					"name":            HostName,
					"description":     "YouTube Music to Discord Rich Presence",
					"path":            e.helperPath(),
					"type":            "stdio",
					"allowed_origins": []any{"chrome-extension://" + id + "/"},
				}
				if b, _ := json.Marshal(m); string(b) != mustJSON(want) {
					t.Fatalf("%s manifest %s", b, data)
				}
			}
			for _, b := range []string{"Edge", "Chromium"} {
				if _, err := os.Stat(e.manifestFile(b)); !os.IsNotExist(err) {
					t.Fatalf("%s is not installed; no manifest expected", b)
				}
			}
			if got, _ := os.ReadFile(e.helperPath()); string(got) != "program v1" {
				t.Fatalf("helper copy %q", got)
			}
			if fi, _ := os.Stat(e.helperPath()); fi.Mode().Perm()&0o111 == 0 {
				t.Fatal("helper is not executable")
			}
			if got, _ := os.ReadFile(filepath.Join(e.dataDir, "config.json")); string(got) != "{\n  \"clientId\": \"123456789012345678\"\n}\n" {
				t.Fatalf("config %q", got)
			}
			if got, _ := os.ReadFile(filepath.Join(e.dataDir, "extension", "icons", "icon-16.png")); string(got) != "png" {
				t.Fatal("extension not unpacked")
			}
			out := e.stdout.String()
			for _, want := range []string{
				"ลงทะเบียนโปรแกรมเสริมกับ Chrome, Brave แล้ว",
				"ID ส่วนขยาย: " + id,
				"1. เปิด chrome://extensions แล้วเปิด Developer mode",
				"2. กด Load unpacked แล้วเลือกโฟลเดอร์ " + filepath.Join(e.dataDir, "extension"),
				"3. เปิดแอป Discord แล้วเล่นเพลงที่ music.youtube.com",
			} {
				if !strings.Contains(out, want) {
					t.Errorf("output lacks %q:\n%s", want, out)
				}
			}
			if len(e.opened) != 0 || strings.Contains(out, "กด Enter") {
				t.Error("with an ID argument nothing opens and nothing waits")
			}
		})
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestNoBrowserFails(t *testing.T) {
	e := newEnv(t, "darwin")
	if code := e.run("", "123456789012345678"); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(e.stderr.String(), "ไม่พบ Chrome บนเครื่องนี้ — ติดตั้ง Chrome ก่อนแล้วรันใหม่") {
		t.Fatalf("stderr %s", e.stderr.String())
	}
}

func TestBadApplicationID(t *testing.T) {
	e := newEnv(t, "darwin", "Chrome")
	for _, id := range []string{"12345", "abc456789012345678", "123456789012345678901"} {
		if code := e.run("", id); code != 1 {
			t.Fatalf("%s: exit %d", id, code)
		}
		if !strings.Contains(e.stderr.String(), "Application ID ต้องเป็นตัวเลข 17–20 หลัก") {
			t.Fatalf("stderr %s", e.stderr.String())
		}
	}
	if _, err := os.Stat(e.dataDir); !os.IsNotExist(err) {
		t.Fatal("nothing should be written for a bad ID")
	}
}

func TestDoubleClickAsksAgainOpensAndWaits(t *testing.T) {
	e := newEnv(t, "darwin", "Chrome")
	if code := e.run("not an id\n 123456789012345678 \r\n\n"); code != 0 {
		t.Fatalf("exit %d\n%s%s", code, e.stdout.String(), e.stderr.String())
	}
	out := e.stdout.String()
	if strings.Count(out, "วาง Application ID จาก Discord Developer Portal: ") != 2 {
		t.Fatalf("should ask twice:\n%s", out)
	}
	if !strings.Contains(e.stderr.String(), "Application ID ต้องเป็นตัวเลข") {
		t.Fatal("bad input should be explained")
	}
	if !strings.HasSuffix(out, "กด Enter เพื่อปิดหน้าต่างนี้") {
		t.Fatalf("should wait for Enter:\n%s", out)
	}
	if len(e.opened) != 1 || e.opened[0] != filepath.Join(e.dataDir, "extension") {
		t.Fatalf("opened %v", e.opened)
	}
}

func TestDoubleClickWithClosedInputStops(t *testing.T) {
	e := newEnv(t, "darwin", "Chrome")
	if code := e.run(""); code != 1 {
		t.Fatalf("exit %d", code)
	}
}

func TestSecondRunReusesIDAndShowsUpdateSteps(t *testing.T) {
	e := newEnv(t, "darwin", "Chrome")
	e.run("", "123456789012345678")
	stale := filepath.Join(e.dataDir, "extension", "old-script.js")
	os.WriteFile(stale, []byte("old"), 0o644)
	os.WriteFile(e.exe, []byte("program v2"), 0o755)

	if code := e.run("\n"); code != 0 {
		t.Fatalf("exit %d\n%s%s", code, e.stdout.String(), e.stderr.String())
	}
	out := e.stdout.String()
	for _, want := range []string{
		"ใช้ Application ID เดิม: 123456789012345678",
		"1. เปิด chrome://extensions แล้วกด ↻ บนการ์ด YouTube Music → Discord",
		"2. refresh แท็บ YouTube Music",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "วาง Application ID") {
		t.Error("should not ask again")
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("files no longer in the extension should be removed")
	}
	if got, _ := os.ReadFile(e.helperPath()); string(got) != "program v2" {
		t.Errorf("helper not replaced: %q", got)
	}
}

func TestNewIDArgumentReplacesTheSavedOne(t *testing.T) {
	e := newEnv(t, "darwin", "Chrome")
	e.run("", "123456789012345678")
	e.run("", "98765432109876543")
	if got, _ := os.ReadFile(filepath.Join(e.dataDir, "config.json")); !strings.Contains(string(got), "98765432109876543") {
		t.Fatalf("config %s", got)
	}
}

func TestUpgradeFromNodeVersionKeepsTheID(t *testing.T) {
	e := newEnv(t, "darwin", "Chrome")
	project := filepath.Join(filepath.Dir(e.dataDir), "my project")
	os.MkdirAll(filepath.Join(project, "host"), 0o755)
	os.WriteFile(filepath.Join(project, "host", "config.json"), []byte(`{"clientId":"111111111111111111"}`), 0o644)
	os.MkdirAll(e.dataDir, 0o755)
	launcher := filepath.Join(e.dataDir, "run-host.sh")
	os.WriteFile(launcher, []byte("#!/bin/sh\nexec \"/usr/local/bin/node\" --disable-warning=ExperimentalWarning \""+
		filepath.Join(project, "host", "main.ts")+"\" \"$@\"\n"), 0o755)

	if code := e.run("\n"); code != 0 {
		t.Fatalf("exit %d\n%s%s", code, e.stdout.String(), e.stderr.String())
	}
	if !strings.Contains(e.stdout.String(), "ใช้ Application ID เดิม: 111111111111111111") {
		t.Fatalf("output:\n%s", e.stdout.String())
	}
	if !strings.Contains(e.stdout.String(), "ไม่ต้องใช้ Node.js") {
		t.Error("should say Node.js is no longer needed")
	}
	if _, err := os.Stat(launcher); !os.IsNotExist(err) {
		t.Error("old launcher should be removed")
	}
}

func TestLegacyWindowsLauncher(t *testing.T) {
	dir := t.TempDir()
	project := filepath.Join(dir, "proj")
	os.MkdirAll(filepath.Join(project, "host"), 0o755)
	os.WriteFile(filepath.Join(project, "host", "config.json"), []byte(`{"clientId":"222222222222222222"}`), 0o644)
	bat := "@echo off\r\n\"C:\\Program Files\\nodejs\\node.exe\" --disable-warning=ExperimentalWarning \"" +
		filepath.Join(project, "host", "main.ts") + "\" %*\r\n"
	os.WriteFile(filepath.Join(dir, "run-host.bat"), []byte(bat), 0o644)
	if got := legacyClientID(dir); got != "222222222222222222" {
		t.Fatalf("got %q", got)
	}
}

func TestUninstall(t *testing.T) {
	e := newEnv(t, "linux", "Chrome", "Chromium")
	e.run("", "123456789012345678")
	if code := e.run("", "--uninstall"); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(e.stdout.String(), "ถอนการลงทะเบียนแล้ว (ลบส่วนขยายเองได้ที่ chrome://extensions)") {
		t.Fatalf("output %s", e.stdout.String())
	}
	for _, b := range []string{"Chrome", "Chromium"} {
		if _, err := os.Stat(e.manifestFile(b)); !os.IsNotExist(err) {
			t.Errorf("%s manifest still there", b)
		}
		if _, err := os.Stat(filepath.Dir(filepath.Dir(e.manifestFile(b)))); err != nil {
			t.Errorf("%s profile folder must stay", b)
		}
	}
	if _, err := os.Stat(e.dataDir); !os.IsNotExist(err) {
		t.Error("data folder still there")
	}
	// Uninstalling twice is fine.
	if code := e.run("", "--uninstall"); code != 0 {
		t.Fatalf("second uninstall exit %d", code)
	}
}

func TestRunningTheInstalledCopyDoesNotCopyOntoItself(t *testing.T) {
	e := newEnv(t, "darwin", "Chrome")
	e.run("", "123456789012345678")
	e.exe = e.helperPath()
	if code := e.run("", "123456789012345678"); code != 0 {
		t.Fatalf("exit %d\n%s", code, e.stderr.String())
	}
	if got, _ := os.ReadFile(e.helperPath()); string(got) != "program v1" {
		t.Fatalf("helper %q", got)
	}
}

func TestDataDirPerOS(t *testing.T) {
	none := func(string) string { return "" }
	cases := []struct {
		goos string
		env  map[string]string
		want string
	}{
		{"darwin", nil, filepath.Join("/Users/u", "Library", "Application Support", "ytm-discord-presence")},
		{"linux", nil, filepath.Join("/Users/u", ".local", "share", "ytm-discord-presence")},
		{"linux", map[string]string{"XDG_DATA_HOME": "/xdg"}, filepath.Join("/xdg", "ytm-discord-presence")},
		{"windows", map[string]string{"LOCALAPPDATA": `C:\Users\u\AppData\Local`}, filepath.Join(`C:\Users\u\AppData\Local`, "ytm-discord-presence")},
		{"darwin", map[string]string{EnvDataDir: "/override"}, "/override"},
	}
	for _, c := range cases {
		getenv := none
		if c.env != nil {
			getenv = func(k string) string { return c.env[k] }
		}
		if got := DataDir(c.goos, "/Users/u", getenv); got != c.want {
			t.Errorf("%s %v: %s, want %s", c.goos, c.env, got, c.want)
		}
	}
}

func TestWindowsTargetsAndOverride(t *testing.T) {
	got := targets("windows", "", func(string) string { return "" })
	if got[0].RegistryKey != `Software\Google\Chrome\NativeMessagingHosts\local.ytm_discord_presence` ||
		got[1].RegistryKey != `Software\Microsoft\Edge\NativeMessagingHosts\local.ytm_discord_presence` {
		t.Fatalf("targets %+v", got)
	}
	got = targets("windows", "", func(k string) string {
		if k == EnvRegistryKey {
			return `Software\ytmd-test`
		}
		return ""
	})
	if got[0].RegistryKey != `Software\ytmd-test\Google\Chrome\NativeMessagingHosts\local.ytm_discord_presence` {
		t.Fatalf("override %+v", got)
	}
}
