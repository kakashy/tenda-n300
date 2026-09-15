package main

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

// captureStderr runs fn with os.Stderr redirected to a pipe and returns
// everything fn wrote to stderr.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	out := make(chan string, 1)
	go func() {
		var sb strings.Builder
		io.Copy(&sb, r)
		out <- sb.String()
	}()
	defer func() {
		os.Stderr = old
		r.Close()
	}()
	fn()
	w.Close()
	return <-out
}

func TestExtractGlobalFlags(t *testing.T) {
	oldJSON, oldProfile := jsonOutput, profileFlag
	t.Cleanup(func() { jsonOutput, profileFlag = oldJSON, oldProfile })

	jsonOutput = false
	profileFlag = ""
	rest, err := extractGlobalFlags([]string{"set", "password", "foo", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if !jsonOutput {
		t.Fatal("expected jsonOutput to be true after --json")
	}
	if got := strings.Join(rest, " "); got != "set password foo" {
		t.Fatalf("got %q, want \"set password foo\"", got)
	}

	jsonOutput = false
	profileFlag = ""
	rest, err = extractGlobalFlags([]string{"--profile", "work", "list"})
	if err != nil {
		t.Fatal(err)
	}
	if profileFlag != "work" {
		t.Fatalf("expected profileFlag work, got %q", profileFlag)
	}
	if got := strings.Join(rest, " "); got != "list" {
		t.Fatalf("got %q, want \"list\"", got)
	}

	profileFlag = ""
	rest, err = extractGlobalFlags([]string{"--profile=work", "list"})
	if err != nil {
		t.Fatal(err)
	}
	if profileFlag != "work" {
		t.Fatalf("expected profileFlag work, got %q", profileFlag)
	}

	// A trailing --profile with no value is an error, not silently dropped.
	if _, err := extractGlobalFlags([]string{"list", "--profile"}); err == nil {
		t.Fatal("expected error for --profile with no value")
	}
	// A following flag must not be swallowed as the --profile value.
	if _, err := extractGlobalFlags([]string{"--profile", "--json", "list"}); err == nil {
		t.Fatal("expected error when --profile value looks like a flag")
	}
}

func TestValidateCommandArgs(t *testing.T) {
	tests := []struct {
		cmd     string
		args    []string
		wantErr bool
	}{
		{"block", []string{"aa:bb:cc:dd:ee:ff"}, false},
		{"block", nil, true},
		{"unblock", []string{"aa:bb:cc:dd:ee:ff"}, false},
		{"unblock", nil, true},
		{"restore", []string{"RouterCfm.cfg"}, false},
		{"restore", nil, true},
		{"portforward", nil, false},
		{"portforward", []string{"list"}, false},
		{"portforward", []string{"add", "192.168.1.10", "22", "22"}, false},
		{"portforward", []string{"add", "192.168.1.10", "22", "22", "--protocol", "tcp"}, false},
		{"portforward", []string{"add", "192.168.1.10", "22"}, true},
		{"portforward", []string{"remove", "2"}, false},
		{"portforward", []string{"remove"}, true},
		{"portforward", []string{"bogus"}, true},
		{"devices", nil, false},
		{"restart", nil, false},
		{"status", []string{"extra"}, false},
	}
	for _, tc := range tests {
		err := validateCommandArgs(tc.cmd, tc.args)
		if (err != nil) != tc.wantErr {
			t.Errorf("validateCommandArgs(%q, %v) = %v, wantErr %v", tc.cmd, tc.args, err, tc.wantErr)
		}
	}
}

func TestConfigSetPasswordTrailingJSONFlag(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true
	cmdProfile([]string{"add", "home", "--ip", "192.168.0.1"})

	jsonOutput = false
	errOut := captureStderr(t, func() {
		cmdConfig([]string{"set", "password", "foo", "--json"})
	})

	pw, err := keyringGetPassword("home")
	if err != nil {
		t.Fatalf("keyringGetPassword(home): %v", err)
	}
	if pw != "foo" {
		t.Fatalf("expected stored password \"foo\", got %q", pw)
	}
	// --json was honored during the call, so the human-readable note (guarded
	// by !jsonOutput) must not have been written.
	if strings.Contains(errOut, "password saved") {
		t.Fatalf("expected no human note in JSON mode, got %q", errOut)
	}
}

func TestConfigShowHonorsTrailingJSONFlag(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true
	cmdProfile([]string{"add", "home", "--ip", "192.168.0.1"})

	jsonOutput = false
	out := captureStdout(t, func() {
		cmdConfig([]string{"--json"})
	})
	if !strings.Contains(out, `"home"`) {
		t.Fatalf("expected JSON config containing home, got %q", out)
	}
}

func TestConfigSetPasswordCreatesDefaultProfile(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true

	cmdConfig([]string{"set", "password", "s3cret"})

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultProfile != "default" {
		t.Fatalf("expected default_profile \"default\", got %q", cfg.DefaultProfile)
	}
	if _, ok := cfg.Profiles["default"]; !ok {
		t.Fatal("expected a \"default\" profile to be created")
	}
	pw, err := keyringGetPassword("default")
	if err != nil {
		t.Fatalf("keyringGetPassword(default): %v", err)
	}
	if pw != "s3cret" {
		t.Fatalf("expected password \"s3cret\", got %q", pw)
	}
}

func TestConfigSetIPRejectsSurplusArgs(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		setupProfileTest(t)
		jsonOutput = true
		cmdConfig([]string{"set", "ip", "1.2.3.4", "extra"})
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestConfigSetIPRejectsSurplusArgs")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected config set ip to fail on surplus value")
	}
}

func TestProfileRemoveDefaultPromotesRemaining(t *testing.T) {
	setupProfileTest(t)
	jsonOutput = true

	cmdProfile([]string{"add", "zeta", "--ip", "10.0.0.2"})
	cmdProfile([]string{"add", "alpha", "--ip", "192.168.0.1"})

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultProfile != "zeta" {
		t.Fatalf("expected first-added profile zeta to be default, got %q", cfg.DefaultProfile)
	}

	cmdProfile([]string{"remove", "zeta"})

	cfg, err = LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Profiles["zeta"]; ok {
		t.Fatal("expected zeta profile removed")
	}
	if cfg.DefaultProfile != "alpha" {
		t.Fatalf("expected remaining profile promoted to default (alpha), got %q", cfg.DefaultProfile)
	}
}

func TestResolveTarget(t *testing.T) {
	setupProfileTest(t)
	keyring.MockInit()

	fpCalls := 0
	oldFP := networkFingerprint
	networkFingerprint = func() (NetworkFingerprint, error) {
		fpCalls++
		return NetworkFingerprint{Gateway: "192.168.0.1", GatewayMAC: "aa:bb:cc:dd:ee:ff"}, nil
	}
	t.Cleanup(func() { networkFingerprint = oldFP })

	cfg := &Config{
		DefaultProfile: "home",
		Profiles: map[string]Profile{
			"home": {IP: "192.168.0.1"},
			"work": {IP: "10.0.0.2"},
		},
		NetworkCache: map[string]string{"192.168.0.1|aa:bb:cc:dd:ee:ff": "work"},
	}

	oldProfile := profileFlag
	t.Cleanup(func() { profileFlag = oldProfile })

	t.Run("fingerprint computed once", func(t *testing.T) {
		fpCalls = 0
		if _, err := resolveTarget("", "", cfg); err != nil {
			t.Fatal(err)
		}
		if fpCalls != 1 {
			t.Fatalf("expected fingerprint computed exactly once, got %d calls", fpCalls)
		}
	})

	t.Run("cache hit detects profile", func(t *testing.T) {
		tr, err := resolveTarget("", "", cfg)
		if err != nil {
			t.Fatal(err)
		}
		if tr.ip != "10.0.0.2" || tr.profileName != "work" {
			t.Fatalf("got (%s, %q), want (10.0.0.2, work)", tr.ip, tr.profileName)
		}
		if !tr.detected || !tr.cacheHit {
			t.Fatalf("expected detected+cacheHit, got detected=%v cacheHit=%v", tr.detected, tr.cacheHit)
		}
	})

	t.Run("cache miss falls back to default", func(t *testing.T) {
		cfg2 := &Config{
			DefaultProfile: "home",
			Profiles: map[string]Profile{
				"home": {IP: "192.168.0.1"},
				"work": {IP: "10.0.0.2"},
			},
			NetworkCache: map[string]string{},
		}
		tr, err := resolveTarget("", "", cfg2)
		if err != nil {
			t.Fatal(err)
		}
		if tr.ip != "192.168.0.1" || tr.profileName != "home" {
			t.Fatalf("got (%s, %q), want (192.168.0.1, home)", tr.ip, tr.profileName)
		}
	})

	t.Run("explicit profile wins over detection", func(t *testing.T) {
		profileFlag = "home"
		defer func() { profileFlag = "" }()
		tr, err := resolveTarget("", "", cfg)
		if err != nil {
			t.Fatal(err)
		}
		if tr.ip != "192.168.0.1" || tr.profileName != "home" {
			t.Fatalf("got (%s, %q), want (192.168.0.1, home)", tr.ip, tr.profileName)
		}
		if tr.detected {
			t.Fatal("expected detected=false for explicit profile")
		}
	})

	t.Run("unknown explicit profile errors", func(t *testing.T) {
		profileFlag = "nope"
		defer func() { profileFlag = "" }()
		_, err := resolveTarget("", "", cfg)
		if err == nil || !strings.Contains(err.Error(), "unknown profile") {
			t.Fatalf("expected unknown profile error, got %v", err)
		}
	})

	t.Run("ip with stored keyring password resolves profile", func(t *testing.T) {
		if err := keyringSetPassword("home", "pw"); err != nil {
			t.Fatal(err)
		}
		tr, err := resolveTarget("192.168.0.1", "", cfg)
		if err != nil {
			t.Fatal(err)
		}
		if tr.profileName != "home" {
			t.Fatalf("expected profile home for keyring lookup, got %q", tr.profileName)
		}
	})

	t.Run("ip with explicit unknown profile errors", func(t *testing.T) {
		profileFlag = "nope"
		defer func() { profileFlag = "" }()
		_, err := resolveTarget("192.168.0.1", "", cfg)
		if err == nil || !strings.Contains(err.Error(), "unknown profile") {
			t.Fatalf("expected unknown profile error, got %v", err)
		}
	})

	t.Run("ip without profile and no default is tolerated", func(t *testing.T) {
		cfg2 := &Config{
			Profiles: map[string]Profile{
				"a": {IP: "1.1.1.1"},
				"b": {IP: "2.2.2.2"},
			},
			NetworkCache: map[string]string{},
		}
		tr, err := resolveTarget("192.168.0.1", "", cfg2)
		if err != nil {
			t.Fatal(err)
		}
		if tr.profileName != "" {
			t.Fatalf("expected no profile resolved, got %q", tr.profileName)
		}
	})
}
