package main

import (
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tobsatop/discord-activity/internal/activity"
	"github.com/tobsatop/discord-activity/internal/discordipc"
	"github.com/tobsatop/discord-activity/internal/fakediscord"
	"github.com/tobsatop/discord-activity/internal/installer"
	"github.com/tobsatop/discord-activity/internal/nativemsg"
)

const runMainEnv = "YTMD_TEST_RUN_MAIN"

func TestMain(m *testing.M) {
	// The test binary doubles as the real program: tests start it like Chrome
	// (or a user) would.
	if os.Getenv(runMainEnv) == "1" {
		os.Exit(run(os.Args[1:]))
	}
	// Safety net: nothing a test starts may touch the real data folder,
	// browser folders, registry keys or Discord app.
	tmp, err := os.MkdirTemp("", "ytmd-main-")
	if err != nil {
		panic(err)
	}
	os.Setenv(installer.EnvDataDir, filepath.Join(tmp, "data"))
	os.Setenv(installer.EnvBrowserHome, filepath.Join(tmp, "home"))
	os.Setenv(installer.EnvRegistryKey, `Software\ytmd-test-unused`)
	os.Setenv(discordipc.EnvIPC, filepath.Join(tmp, "no-discord"))
	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

func extensionFS(t *testing.T) fs.FS {
	t.Helper()
	sub, err := fs.Sub(embedded, "extension")
	if err != nil {
		t.Fatal(err)
	}
	return sub
}

func TestVersionsMatch(t *testing.T) {
	var manifest, pkg struct{ Version string }
	data, _ := fs.ReadFile(extensionFS(t), "manifest.json")
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("package.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != version || pkg.Version != version {
		t.Fatalf("version.go %s, extension/manifest.json %s, package.json %s", version, manifest.Version, pkg.Version)
	}
}

func TestEmbeddedExtensionIsComplete(t *testing.T) {
	ext := extensionFS(t)
	data, _ := fs.ReadFile(ext, "manifest.json")
	var m struct {
		Background struct {
			ServiceWorker string `json:"service_worker"`
		}
		ContentScripts []struct{ JS []string } `json:"content_scripts"`
		Icons          map[string]string
		Action         struct {
			DefaultIcon map[string]string `json:"default_icon"`
		}
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	need := []string{m.Background.ServiceWorker}
	for _, cs := range m.ContentScripts {
		need = append(need, cs.JS...)
	}
	for _, p := range m.Icons {
		need = append(need, p)
	}
	for _, p := range m.Action.DefaultIcon {
		need = append(need, p)
	}
	// Toolbar states set by background.js: icons/<state>-16|32.png
	for _, state := range []string{"icon", "idle", "warn", "error"} {
		need = append(need, "icons/"+state+"-16.png", "icons/"+state+"-32.png")
	}
	for _, p := range need {
		if _, err := fs.Stat(ext, p); err != nil {
			t.Errorf("the embedded extension is missing %s", p)
		}
	}
	bg, _ := fs.ReadFile(ext, "background.js")
	if !strings.Contains(string(bg), "const HOST_NAME = '"+installer.HostName+"';") {
		t.Errorf("HOST_NAME in background.js differs from %s", installer.HostName)
	}
}

func TestExtensionIDFromKey(t *testing.T) {
	data, _ := fs.ReadFile(extensionFS(t), "manifest.json")
	id, err := installer.ExtensionID(data)
	if err != nil {
		t.Fatal(err)
	}
	if id != "bfcijjaglbmlpnnpnekjcamlmgdkdebb" {
		t.Fatalf("extension id %s", id)
	}
}

func TestHelperLaunchDetection(t *testing.T) {
	cases := map[string]bool{
		"chrome-extension://bfcijjaglbmlpnnpnekjcamlmgdkdebb/": true,
		"--parent-window=0":  true,
		"123456789012345678": false,
		"--uninstall":        false,
	}
	for arg, want := range cases {
		if got := isHelperLaunch([]string{arg}); got != want {
			t.Errorf("%s: %v", arg, got)
		}
	}
	if isHelperLaunch(nil) {
		t.Error("no args is the installer")
	}
}

// ---- helper mode, end to end: the program as Chrome runs it, against a fake Discord.

var trackJSON = map[string]any{
	"videoId":    "dQw4w9WgXcQ",
	"title":      "Song Title",
	"artist":     "Some Artist",
	"album":      "Some Album",
	"artworkUrl": "https://lh3.googleusercontent.com/abc=w544-h544",
	"startMs":    float64(time.Now().UnixMilli() - 30_000),
	"endMs":      float64(time.Now().UnixMilli() + 170_000),
	"playing":    true,
}

func withPlaying(playing bool) map[string]any {
	t := map[string]any{}
	for k, v := range trackJSON {
		t[k] = v
	}
	t["playing"] = playing
	return t
}

type hostProc struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	exited  chan int
	dataDir string

	mu       sync.Mutex
	fromHost []map[string]any
}

func startHost(t *testing.T, ipcBase string, env map[string]string, args ...string) *hostProc {
	t.Helper()
	dataDir := t.TempDir()
	if len(args) == 0 {
		args = []string{"chrome-extension://bfcijjaglbmlpnnpnekjcamlmgdkdebb/"}
	}
	cmd := exec.Command(os.Args[0], args...)
	vars := map[string]string{
		runMainEnv:           "1",
		installer.EnvDataDir: dataDir,
		discordipc.EnvIPC:    ipcBase,
		"DISCORD_CLIENT_ID":  "123456789012345678",
	}
	for k, v := range env {
		vars[k] = v
	}
	cmd.Env = os.Environ()
	for k, v := range vars {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	h := &hostProc{cmd: cmd, stdin: stdin, exited: make(chan int, 1), dataDir: dataDir}
	go func() {
		_ = nativemsg.Read(stdout, func(m any) {
			msg, _ := m.(map[string]any)
			h.mu.Lock()
			h.fromHost = append(h.fromHost, msg)
			h.mu.Unlock()
		}, func(err error) { t.Errorf("bad frame from the helper: %v", err) })
		err := cmd.Wait()
		code := 0
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else if err != nil {
			code = -1
		}
		h.exited <- code
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		select {
		case <-h.exited:
		case <-time.After(5 * time.Second):
		}
	})
	return h
}

func (h *hostProc) update(t *testing.T, track any) {
	t.Helper()
	frame, _ := nativemsg.Encode(map[string]any{"type": "update", "track": track})
	if _, err := h.stdin.Write(frame); err != nil {
		t.Fatal(err)
	}
}

func (h *hostProc) status(state string) (map[string]any, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, m := range h.fromHost {
		if m["type"] == "status" && m["state"] == state {
			return m, true
		}
	}
	return nil, false
}

func (h *hostProc) waitStatus(t *testing.T, state string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if m, ok := h.status(state); ok {
			return m
		}
		if time.Now().After(deadline) {
			h.mu.Lock()
			defer h.mu.Unlock()
			t.Fatalf("no %s status; got %v", state, h.fromHost)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (h *hostProc) log(t *testing.T) string {
	t.Helper()
	data, _ := os.ReadFile(filepath.Join(h.dataDir, "host.log"))
	return string(data)
}

func count(n int) func([]*activity.Activity) bool {
	return func(l []*activity.Activity) bool { return len(l) == n }
}

func TestHelperShowsClearsAndExits(t *testing.T) {
	discord := fakediscord.Start(t, fakediscord.Options{})
	host := startHost(t, discord.Base, nil)

	host.update(t, trackJSON)
	discord.WaitFor(t, count(1))
	a := discord.Activities()[0]
	if a.Details != "Song Title" || a.State != "Some Artist" || a.Type != 2 {
		t.Fatalf("first activity %+v", a)
	}

	host.update(t, withPlaying(false))
	discord.WaitFor(t, count(2))
	if discord.Activities()[1] != nil {
		t.Fatal("pause should clear the activity")
	}

	host.update(t, trackJSON)
	discord.WaitFor(t, count(3))
	host.stdin.Close() // what Chrome does when the last YouTube Music tab closes
	discord.WaitFor(t, count(4))
	if discord.Activities()[3] != nil {
		t.Fatal("disconnect should clear the activity")
	}
	select {
	case code := <-host.exited:
		if code != 0 {
			t.Fatalf("exit code %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("helper did not exit")
	}
	host.waitStatus(t, "connected")

	log := host.log(t)
	for _, want := range []string{"host started (ytm-discord " + version, "discord: connected", "playing: Song Title — Some Artist", "Chrome disconnected; clearing the status"} {
		if !strings.Contains(log, want) {
			t.Errorf("host.log lacks %q:\n%s", want, log)
		}
	}
	if strings.Contains(log, "tester") {
		t.Errorf("host.log must not contain the Discord user name:\n%s", log)
	}
	if !regexp.MustCompile(`^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{3}Z host started`).MatchString(log) {
		t.Errorf("log line format:\n%s", log)
	}
}

func TestHelperFallsBackToBasicFields(t *testing.T) {
	discord := fakediscord.Start(t, fakediscord.Options{RejectDetailsURL: true})
	host := startHost(t, discord.Base, nil)

	host.update(t, trackJSON)
	discord.WaitFor(t, count(2))
	list := discord.Activities()
	if list[0].DetailsURL == "" {
		t.Fatal("first try has the link")
	}
	if list[1].DetailsURL != "" || list[1].Details != "Song Title" {
		t.Fatalf("retry should drop the link: %+v", list[1])
	}
}

func TestHelperReportsNoDiscordAndNoApplicationID(t *testing.T) {
	nothing := fakediscord.NewBase(t)
	noDiscord := startHost(t, nothing, nil)
	noID := startHost(t, nothing, map[string]string{"DISCORD_CLIENT_ID": ""})

	if m := noDiscord.waitStatus(t, "unavailable"); m["detail"] != "Discord app is not running" {
		t.Fatalf("detail %v", m["detail"])
	}
	m := noID.waitStatus(t, "error")
	if m["detail"] != "ยังไม่ได้ใส่ Discord Application ID — รันตัวติดตั้ง (README ขั้นที่ 3)" {
		t.Fatalf("detail %v", m["detail"])
	}
}

func TestHelperReadsConfigAndWindowsArgs(t *testing.T) {
	discord := fakediscord.Start(t, fakediscord.Options{})
	dataDir := t.TempDir()
	os.WriteFile(filepath.Join(dataDir, "config.json"), []byte(`{"clientId": "123456789012345678"}`), 0o644)
	host := startHost(t, discord.Base, map[string]string{"DISCORD_CLIENT_ID": "", installer.EnvDataDir: dataDir},
		"chrome-extension://bfcijjaglbmlpnnpnekjcamlmgdkdebb/", "--parent-window=0")
	host.dataDir = dataDir
	host.update(t, trackJSON)
	discord.WaitFor(t, count(1))
	host.waitStatus(t, "connected")
}

func TestHelperTrimsABigLog(t *testing.T) {
	dataDir := t.TempDir()
	logFile := filepath.Join(dataDir, "host.log")
	os.WriteFile(logFile, []byte(strings.Repeat("x", 1024*1024+1)), 0o644)
	host := startHost(t, fakediscord.NewBase(t), map[string]string{installer.EnvDataDir: dataDir})
	host.dataDir = dataDir
	host.waitStatus(t, "unavailable")
	fi, err := os.Stat(logFile)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() > 4096 || !strings.Contains(host.log(t), "host started") {
		t.Fatalf("log not trimmed: %d bytes", fi.Size())
	}
}

func TestHelperIgnoresJunk(t *testing.T) {
	discord := fakediscord.Start(t, fakediscord.Options{})
	host := startHost(t, discord.Base, nil)
	host.waitStatus(t, "connected")
	bad := []byte{5, 0, 0, 0, '{', 'n', 'o', 'p', 'e'}
	host.stdin.Write(bad)
	other, _ := nativemsg.Encode(map[string]any{"type": "hello"})
	host.stdin.Write(other)
	host.update(t, trackJSON)
	discord.WaitFor(t, count(1))
	if !strings.Contains(host.log(t), "bad message from Chrome:") {
		t.Errorf("junk not logged:\n%s", host.log(t))
	}
}

// ---- installer mode, as a user runs it (with every location overridden).

func runProgram(t *testing.T, env map[string]string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), runMainEnv+"=1")
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.CombinedOutput()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return string(out), code
}

func TestInstallAndUninstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("registry install is covered by internal/installer's Windows test and the CI smoke test")
	}
	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data")
	home := filepath.Join(tmp, "home")
	chromeDir := filepath.Join(home, "Library", "Application Support", "Google", "Chrome")
	if runtime.GOOS != "darwin" {
		chromeDir = filepath.Join(home, ".config", "google-chrome")
	}
	os.MkdirAll(chromeDir, 0o755)
	env := map[string]string{installer.EnvDataDir: dataDir, installer.EnvBrowserHome: home}

	out, code := runProgram(t, env, "123456789012345678")
	if code != 0 {
		t.Fatalf("install failed (%d):\n%s", code, out)
	}
	if !strings.Contains(out, "ลงทะเบียนโปรแกรมเสริมกับ Chrome แล้ว") || !strings.Contains(out, filepath.Join(dataDir, "extension")) {
		t.Fatalf("output:\n%s", out)
	}

	manifestFile := filepath.Join(chromeDir, "NativeMessagingHosts", installer.HostName+".json")
	var manifest struct {
		Name, Path, Type string
		AllowedOrigins   []string `json:"allowed_origins"`
	}
	data, err := os.ReadFile(manifestFile)
	if err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(data, &manifest)
	helperPath := filepath.Join(dataDir, installer.HelperName(runtime.GOOS))
	if manifest.Name != installer.HostName || manifest.Path != helperPath || manifest.Type != "stdio" ||
		len(manifest.AllowedOrigins) != 1 || manifest.AllowedOrigins[0] != "chrome-extension://bfcijjaglbmlpnnpnekjcamlmgdkdebb/" {
		t.Fatalf("manifest %s", data)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "extension", "manifest.json")); err != nil {
		t.Fatal("extension not unpacked")
	}

	// The installed copy is the same program: Chrome can run it as the helper.
	cmd := exec.Command(helperPath, "--version")
	cmd.Env = append(os.Environ(), runMainEnv+"=1")
	if got, err := cmd.Output(); err != nil || strings.TrimSpace(string(got)) != version {
		t.Fatalf("installed helper --version: %q %v", got, err)
	}

	out, code = runProgram(t, env, "--uninstall")
	if code != 0 || !strings.Contains(out, "ถอนการลงทะเบียนแล้ว") {
		t.Fatalf("uninstall (%d):\n%s", code, out)
	}
	if _, err := os.Stat(manifestFile); !os.IsNotExist(err) {
		t.Fatal("manifest not removed")
	}
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Fatal("data folder not removed")
	}
}
