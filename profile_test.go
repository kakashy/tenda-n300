package main

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

// setupProfileTest isolates cmdProfile/cmdConfig from the real environment:
//   - HOME is redirected to a temp dir so configDir() resolves to
//     $TMP/.config/tenda-n300 and the real ~/.config is never touched
//     (configDir() in config.go uses os.UserHomeDir(), which on Linux reads
//     $HOME).
//   - The OS keyring is swapped for go-keyring's in-memory mock provider via
//     keyring.MockInit(). The package-level keyringGetPassword/SetPassword/
//     DeletePassword functions cannot be reassigned (they are plain funcs), but
//     they delegate to the keyring package, whose provider var we CAN swap. That
//     exercises the real code paths for per-profile keys ("password" vs
//     "password:<name>") without a real secret service.
//
// The jsonOutput and profileFlag package globals are restored after each test.
func setupProfileTest(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	keyring.MockInit()

	oldJSON := jsonOutput
	oldProfile := profileFlag
	t.Cleanup(func() {
		jsonOutput = oldJSON
		profileFlag = oldProfile
	})
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns
// everything fn wrote to stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	out := make(chan string, 1)
	go func() {
		var sb strings.Builder
		io.Copy(&sb, r)
		out <- sb.String()
	}()
	defer func() {
		os.Stdout = old
		r.Close()
	}()
	fn()
	w.Close()
	return <-out
}

func TestProfileAddListUse(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true

	cmdProfile([]string{"add", "home", "--ip", "192.168.0.1"})

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profiles["home"].IP != "192.168.0.1" {
		t.Fatalf("expected home profile with IP 192.168.0.1, got %+v", cfg.Profiles["home"])
	}

	out := captureStdout(t, func() {
		cmdProfile([]string{"list"})
	})
	if !strings.Contains(out, "home") {
		t.Fatalf("profile list output %q does not contain \"home\"", out)
	}

	cmdProfile([]string{"use", "home"})

	cfg, err = LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultProfile != "home" {
		t.Fatalf("expected default profile \"home\", got %q", cfg.DefaultProfile)
	}
}

func TestProfileUseSwitchesDefault(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true

	cmdProfile([]string{"add", "home", "--ip", "192.168.0.1"})
	cmdProfile([]string{"add", "work", "--ip", "10.0.0.1"})

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultProfile != "home" {
		t.Fatalf("expected default \"home\" (first profile), got %q", cfg.DefaultProfile)
	}

	cmdProfile([]string{"use", "work"})

	cfg, err = LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultProfile != "work" {
		t.Fatalf("expected default \"work\" after profile use, got %q", cfg.DefaultProfile)
	}
	if _, ok := cfg.Profiles["home"]; !ok {
		t.Fatal("expected home profile to still exist")
	}
}

func TestConfigSetIPScopedToActiveProfile(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true

	cmdProfile([]string{"add", "home", "--ip", "192.168.0.1"})
	cmdProfile([]string{"add", "work", "--ip", "10.0.0.2"})

	cmdConfig([]string{"set", "ip", "10.0.0.1"})

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profiles["home"].IP != "10.0.0.1" {
		t.Fatalf("expected active profile home IP updated to 10.0.0.1, got %q", cfg.Profiles["home"].IP)
	}
	if cfg.Profiles["work"].IP != "10.0.0.2" {
		t.Fatalf("expected work profile IP untouched, got %q", cfg.Profiles["work"].IP)
	}
}

func TestConfigSetIPScopedViaProfileFlag(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true

	cmdProfile([]string{"add", "home", "--ip", "192.168.0.1"})
	cmdProfile([]string{"add", "work", "--ip", "10.0.0.2"})

	profileFlag = "work"
	cmdConfig([]string{"set", "ip", "10.0.0.3"})
	profileFlag = ""

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profiles["work"].IP != "10.0.0.3" {
		t.Fatalf("expected work profile IP updated to 10.0.0.3, got %q", cfg.Profiles["work"].IP)
	}
	if cfg.Profiles["home"].IP != "192.168.0.1" {
		t.Fatalf("expected home profile IP untouched, got %q", cfg.Profiles["home"].IP)
	}
}

func TestConfigSetIPCreatesDefaultProfile(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true

	cmdConfig([]string{"set", "ip", "10.1.1.1"})

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultProfile != "default" {
		t.Fatalf("expected default_profile \"default\", got %q", cfg.DefaultProfile)
	}
	if cfg.Profiles["default"].IP != "10.1.1.1" {
		t.Fatalf("expected default profile IP 10.1.1.1, got %q", cfg.Profiles["default"].IP)
	}
}

func TestConfigSetPasswordPerProfile(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true

	cmdProfile([]string{"add", "home", "--ip", "192.168.0.1"})
	cmdConfig([]string{"set", "password", "s3cret"})

	pw, err := keyringGetPassword("home")
	if err != nil {
		t.Fatalf("keyringGetPassword(home): %v", err)
	}
	if pw != "s3cret" {
		t.Fatalf("expected password \"s3cret\", got %q", pw)
	}

	// The password must live under the per-profile key "password:home", not the
	// bare "password" key that belongs to the "default" profile.
	if _, err := keyringGetPassword(""); err == nil {
		t.Fatal("expected no password stored under the default (bare) key")
	}
	if _, err := keyringGetPassword("default"); err == nil {
		t.Fatal("expected no password stored under the \"default\" profile key")
	}
}

func TestProfileRename(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true

	cmdProfile([]string{"add", "home", "--ip", "192.168.0.1"})
	cmdProfile([]string{"add", "work", "--ip", "10.0.0.2"})

	cmdProfile([]string{"rename", "home", "hq"})

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Profiles["home"]; ok {
		t.Fatal("expected old profile name removed after rename")
	}
	if cfg.Profiles["hq"].IP != "192.168.0.1" {
		t.Fatalf("expected renamed profile hq with IP 192.168.0.1, got %+v", cfg.Profiles["hq"])
	}
	if cfg.DefaultProfile != "hq" {
		t.Fatalf("expected default profile renamed to hq, got %q", cfg.DefaultProfile)
	}
}

func TestProfileRemove(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true

	cmdProfile([]string{"add", "home", "--ip", "192.168.0.1"})
	cmdProfile([]string{"add", "work", "--ip", "10.0.0.2"})

	cmdProfile([]string{"remove", "work"})
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Profiles["work"]; ok {
		t.Fatal("expected work profile removed")
	}
	if cfg.DefaultProfile != "home" {
		t.Fatalf("expected default still \"home\", got %q", cfg.DefaultProfile)
	}

	cmdProfile([]string{"remove", "home"})
	cfg, err = LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Profiles) != 0 {
		t.Fatalf("expected no profiles remaining, got %+v", cfg.Profiles)
	}
	if cfg.DefaultProfile != "" {
		t.Fatalf("expected empty default profile, got %q", cfg.DefaultProfile)
	}
}

func TestParseProfileFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantIP   string
		wantPass string
		wantRest []string
		wantErr  bool
	}{
		{
			name:     "flags after positional",
			args:     []string{"home", "--ip", "192.168.0.1", "--password", "pw1"},
			wantIP:   "192.168.0.1",
			wantPass: "pw1",
			wantRest: []string{"home"},
		},
		{
			name:     "equals form",
			args:     []string{"--ip=10.0.0.1", "home", "--password=pw2"},
			wantIP:   "10.0.0.1",
			wantPass: "pw2",
			wantRest: []string{"home"},
		},
		{
			name:     "no flags",
			args:     []string{"home"},
			wantRest: []string{"home"},
		},
		{
			name:    "flag at end without value",
			args:    []string{"home", "--ip"},
			wantErr: true,
		},
		{
			name:    "missing value does not swallow next flag",
			args:    []string{"home", "--password", "--ip", "10.0.0.1"},
			wantErr: true,
		},
		{
			name:     "double dash terminates flags",
			args:     []string{"--password", "pw", "--", "--ip", "9.9.9.9"},
			wantPass: "pw",
			wantRest: []string{"--ip", "9.9.9.9"},
		},
		{
			name:     "empty equals value",
			args:     []string{"--ip=", "home"},
			wantIP:   "",
			wantRest: []string{"home"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip, pass, rest, err := parseProfileFlags(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ip != tc.wantIP || pass != tc.wantPass {
				t.Fatalf("got (ip=%q, pass=%q), want (ip=%q, pass=%q)", ip, pass, tc.wantIP, tc.wantPass)
			}
			if len(rest) != len(tc.wantRest) {
				t.Fatalf("got rest %q, want %q", rest, tc.wantRest)
			}
			for i := range rest {
				if rest[i] != tc.wantRest[i] {
					t.Fatalf("got rest %q, want %q", rest, tc.wantRest)
				}
			}
		})
	}
}

func TestProfileSubcommandJSONFlag(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true
	cmdProfile([]string{"add", "home", "--ip", "192.168.0.1"})

	jsonOutput = false
	out := captureStdout(t, func() {
		cmdProfile([]string{"list", "--json"})
	})
	if !strings.Contains(out, `"home"`) {
		t.Fatalf("expected JSON profile list containing home, got %q", out)
	}
}

func TestProfileRemoveCleansNetworkCache(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true
	cmdProfile([]string{"add", "home", "--ip", "192.168.0.1"})
	cmdProfile([]string{"add", "work", "--ip", "10.0.0.2"})

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.NetworkCache["192.168.0.1|aa:bb:cc:dd:ee:ff"] = "work"
	cfg.NetworkCache["10.0.0.2|aa:bb:cc:dd:ee:fe"] = "home"
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	cmdProfile([]string{"remove", "work"})

	cfg, err = LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.NetworkCache["192.168.0.1|aa:bb:cc:dd:ee:ff"]; ok {
		t.Fatal("expected cache entry pointing at removed profile to be deleted")
	}
	if cfg.NetworkCache["10.0.0.2|aa:bb:cc:dd:ee:fe"] != "home" {
		t.Fatal("expected unrelated cache entry to remain")
	}
}

func TestProfileRenameRewritesNetworkCache(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true
	cmdProfile([]string{"add", "home", "--ip", "192.168.0.1"})
	cmdProfile([]string{"add", "work", "--ip", "10.0.0.2"})

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.NetworkCache["192.168.0.1|aa:bb:cc:dd:ee:ff"] = "home"
	cfg.NetworkCache["10.0.0.2|aa:bb:cc:dd:ee:fe"] = "work"
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	cmdProfile([]string{"rename", "home", "hq"})

	cfg, err = LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NetworkCache["192.168.0.1|aa:bb:cc:dd:ee:ff"] != "hq" {
		t.Fatalf("expected cache entry rewritten to hq, got %q", cfg.NetworkCache["192.168.0.1|aa:bb:cc:dd:ee:ff"])
	}
	if cfg.NetworkCache["10.0.0.2|aa:bb:cc:dd:ee:fe"] != "work" {
		t.Fatal("expected other cache entry untouched")
	}
}

func TestProfileShowWithNoDefault(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true

	cfg := &Config{
		Profiles: map[string]Profile{
			"home": {IP: "192.168.0.1"},
			"work": {IP: "10.0.0.2"},
		},
		NetworkCache: map[string]string{},
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		cmdProfile([]string{})
	})
	if !strings.Contains(out, `"home"`) || !strings.Contains(out, `"work"`) {
		t.Fatalf("expected profile list in output, got %q", out)
	}
	if strings.Contains(out, `"active": "home"`) || strings.Contains(out, `"active": "work"`) {
		t.Fatalf("expected no active profile when default is unset, got %q", out)
	}
}

func TestConfigShowWithNoDefaultOmitsNetworkCache(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true

	cfg := &Config{
		Profiles: map[string]Profile{
			"home": {IP: "192.168.0.1"},
			"work": {IP: "10.0.0.2"},
		},
		NetworkCache: map[string]string{"10.0.0.2|aa:bb:cc:dd:ee:ff": "home"},
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		cmdConfig([]string{})
	})
	if !strings.Contains(out, `"home"`) || !strings.Contains(out, `"work"`) {
		t.Fatalf("expected profiles in config show JSON, got %q", out)
	}
	if strings.Contains(out, "network_cache") {
		t.Fatalf("config show JSON must omit network_cache, got %q", out)
	}
}

func TestProfileAddRejectsSurplusArgs(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		setupProfileTest(t)
		jsonOutput = true
		cmdProfile([]string{"add", "home", "--ip", "10.0.0.1", "extra"})
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestProfileAddRejectsSurplusArgs")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected profile add to fail on surplus positional arguments")
	}
}
