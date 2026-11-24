package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows/registry"
)

// Registers under a throwaway HKCU\Software\ytmd-test-… key, never the real browser keys.
func TestWindowsRegistryInstallAndUninstall(t *testing.T) {
	root := fmt.Sprintf(`Software\ytmd-test-%d`, time.Now().UnixNano())
	t.Cleanup(func() { deleteTree(registry.CURRENT_USER, root) })
	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data")
	exe := filepath.Join(tmp, "downloaded.exe")
	os.WriteFile(exe, []byte("program"), 0o755)
	getenv := func(k string) string {
		switch k {
		case EnvDataDir:
			return dataDir
		case EnvRegistryKey:
			return root
		}
		return ""
	}
	var stdout, stderr bytes.Buffer
	opts := Options{
		Args: []string{"123456789012345678"}, Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
		Getenv: getenv, GOOS: "windows", Home: tmp, Executable: exe, Extension: testExtension(),
	}
	if code := Run(opts); code != 0 {
		t.Fatalf("exit %d\n%s%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "ลงทะเบียนโปรแกรมเสริมกับ Chrome, Edge แล้ว") {
		t.Fatalf("output %s", stdout.String())
	}
	manifestFile := filepath.Join(dataDir, "native-host.json")
	for _, tg := range targets("windows", tmp, getenv) {
		k, err := registry.OpenKey(registry.CURRENT_USER, tg.RegistryKey, registry.QUERY_VALUE)
		if err != nil {
			t.Fatalf("%s key: %v", tg.Browser, err)
		}
		v, _, err := k.GetStringValue("")
		k.Close()
		if err != nil || v != manifestFile {
			t.Fatalf("%s default value %q %v", tg.Browser, v, err)
		}
	}
	var m struct{ Path string }
	data, _ := os.ReadFile(manifestFile)
	json.Unmarshal(data, &m)
	if m.Path != filepath.Join(dataDir, "ytm-discord-host.exe") {
		t.Fatalf("manifest %s", data)
	}

	opts.Args = []string{"--uninstall"}
	if code := Run(opts); code != 0 {
		t.Fatalf("uninstall exit %d", code)
	}
	for _, tg := range targets("windows", tmp, getenv) {
		if k, err := registry.OpenKey(registry.CURRENT_USER, tg.RegistryKey, registry.QUERY_VALUE); err == nil {
			k.Close()
			t.Fatalf("%s key still there", tg.Browser)
		}
	}
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Fatal("data folder still there")
	}
}

func deleteTree(root registry.Key, path string) {
	if k, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS); err == nil {
		names, _ := k.ReadSubKeyNames(-1)
		k.Close()
		for _, name := range names {
			deleteTree(root, path+`\`+name)
		}
	}
	_ = registry.DeleteKey(root, path)
}
