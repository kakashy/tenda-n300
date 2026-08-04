package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigLoadSave(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Profiles == nil || cfg.NetworkCache == nil {
		t.Fatal("expected initialized profile and network cache maps")
	}

	cfg.DefaultProfile = "home"
	cfg.Profiles["home"] = Profile{IP: "192.168.1.1"}
	cfg.NetworkCache["192.168.1.1|aa:bb:cc:dd:ee:ff"] = "home"
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultProfile != "home" {
		t.Fatalf("expected default profile home, got %s", loaded.DefaultProfile)
	}
	if loaded.Profiles["home"].IP != "192.168.1.1" {
		t.Fatalf("expected 192.168.1.1, got %s", loaded.Profiles["home"].IP)
	}
	if loaded.NetworkCache["192.168.1.1|aa:bb:cc:dd:ee:ff"] != "home" {
		t.Fatalf("expected cached profile home, got %s", loaded.NetworkCache["192.168.1.1|aa:bb:cc:dd:ee:ff"])
	}
}

func TestConfigFilePermissions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := &Config{
		Profiles:     map[string]Profile{"home": {IP: "10.0.0.1"}},
		NetworkCache: map[string]string{},
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	path, _ := configPath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0177 != 0 {
		t.Fatalf("config file is world-readable: %v", info.Mode())
	}
}

func TestConfigDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	d, err := configDir()
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(dir, ".config", "tenda-n300")
	if d != expected {
		t.Fatalf("expected %s, got %s", expected, d)
	}
	if _, err := os.Stat(d); os.IsNotExist(err) {
		t.Fatal("config dir was not created")
	}
}

func TestConfigDirPermission(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	d, err := configDir()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(d)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("config dir is group/world-accessible: %v", info.Mode())
	}
}

func TestConfigMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultProfile != "" || len(cfg.Profiles) != 0 || len(cfg.NetworkCache) != 0 {
		t.Fatal("expected empty config for missing file")
	}
}

func TestConfigEmptyFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	path, _ := configPath()
	if err := os.WriteFile(path, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profiles == nil || cfg.NetworkCache == nil {
		t.Fatal("expected initialized maps for empty file")
	}
}

func TestLegacyConfigMigration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	path, _ := configPath()
	if err := os.WriteFile(path, []byte(`{"ip":"192.168.0.1"}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultProfile != "default" {
		t.Fatalf("expected DefaultProfile \"default\", got %q", cfg.DefaultProfile)
	}
	if cfg.Profiles["default"].IP != "192.168.0.1" {
		t.Fatalf("expected migrated IP 192.168.0.1, got %q", cfg.Profiles["default"].IP)
	}
	if cfg.NetworkCache == nil {
		t.Fatal("expected non-nil NetworkCache after migration")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("rewritten config is not valid JSON: %v", err)
	}
	if _, ok := raw["profiles"]; !ok {
		t.Fatal("rewritten config is missing the profiles key")
	}
	if _, ok := raw["ip"]; ok {
		t.Fatal("rewritten config still has the legacy ip key")
	}
}

func TestNewFormatLoadNilMaps(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	path, _ := configPath()
	if err := os.WriteFile(path, []byte(`{"default_profile":"work","profiles":{"work":{"ip":"10.1.2.3"}}}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultProfile != "work" {
		t.Fatalf("expected default profile work, got %q", cfg.DefaultProfile)
	}
	if cfg.Profiles["work"].IP != "10.1.2.3" {
		t.Fatalf("expected IP 10.1.2.3, got %q", cfg.Profiles["work"].IP)
	}
	if cfg.NetworkCache == nil {
		t.Fatal("expected non-nil NetworkCache")
	}
}

func TestNewFormatLoadBothMapsNil(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	path, _ := configPath()
	if err := os.WriteFile(path, []byte(`{"profiles":{}}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profiles == nil || cfg.NetworkCache == nil {
		t.Fatal("expected nil maps to be initialized")
	}
}

func TestActiveProfileExplicit(t *testing.T) {
	cfg := &Config{
		DefaultProfile: "home",
		Profiles: map[string]Profile{
			"home": {IP: "192.168.0.1"},
			"work": {IP: "10.0.0.1"},
		},
		NetworkCache: map[string]string{},
	}
	p, name, err := ActiveProfile(cfg, "work")
	if err != nil {
		t.Fatal(err)
	}
	if name != "work" || p.IP != "10.0.0.1" {
		t.Fatalf("got (%+v, %q), want work/10.0.0.1", p, name)
	}
}

func TestActiveProfileMissingName(t *testing.T) {
	cfg := &Config{
		Profiles:     map[string]Profile{"home": {IP: "192.168.0.1"}},
		NetworkCache: map[string]string{},
	}
	if _, _, err := ActiveProfile(cfg, "nope"); err == nil {
		t.Fatal("expected error for unknown profile name")
	}
}

func TestActiveProfileDefault(t *testing.T) {
	cfg := &Config{
		DefaultProfile: "home",
		Profiles: map[string]Profile{
			"home": {IP: "192.168.0.1"},
			"work": {IP: "10.0.0.1"},
		},
		NetworkCache: map[string]string{},
	}
	p, name, err := ActiveProfile(cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if name != "home" || p.IP != "192.168.0.1" {
		t.Fatalf("got (%+v, %q), want home/192.168.0.1", p, name)
	}
}

func TestActiveProfileSingleFallback(t *testing.T) {
	cfg := &Config{
		Profiles:     map[string]Profile{"only": {IP: "192.168.0.1"}},
		NetworkCache: map[string]string{},
	}
	p, name, err := ActiveProfile(cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if name != "only" || p.IP != "192.168.0.1" {
		t.Fatalf("got (%+v, %q), want only/192.168.0.1", p, name)
	}
}

func TestActiveProfileZero(t *testing.T) {
	cfg := &Config{Profiles: map[string]Profile{}, NetworkCache: map[string]string{}}
	p, name, err := ActiveProfile(cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if p != nil || name != "" {
		t.Fatalf("got (%+v, %q), want (nil, \"\") for zero profiles", p, name)
	}
}

func TestActiveProfileAmbiguous(t *testing.T) {
	cfg := &Config{
		Profiles: map[string]Profile{
			"home": {IP: "192.168.0.1"},
			"work": {IP: "10.0.0.1"},
		},
		NetworkCache: map[string]string{},
	}
	_, _, err := ActiveProfile(cfg, "")
	if err == nil {
		t.Fatal("expected error for multiple profiles without a default")
	}
}

func TestActiveProfileNilConfig(t *testing.T) {
	p, name, err := ActiveProfile(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if p != nil || name != "" {
		t.Fatalf("got (%+v, %q), want (nil, \"\") for nil config", p, name)
	}
}

func TestValidateProfileName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"home", false},
		{"work", false},
		{"work-1", false},
		{"a_b", false},
		{"default", false},
		{strings.Repeat("a", 32), false},
		{"", true},
		{"a b", true},
		{"x!", true},
		{strings.Repeat("a", 33), true},
		{"home:2", true}, // colon is reserved for keyring keys
		{"-start", true},
		{"_start", true},
	}
	for _, tc := range tests {
		err := validateProfileName(tc.name)
		if (err != nil) != tc.wantErr {
			t.Errorf("validateProfileName(%q) = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
	}
}

func TestActiveProfileTable(t *testing.T) {
	twoProfiles := func() *Config {
		return &Config{
			DefaultProfile: "home",
			Profiles: map[string]Profile{
				"home": {IP: "192.168.0.1"},
				"work": {IP: "10.0.0.1"},
			},
			NetworkCache: map[string]string{},
		}
	}
	tests := []struct {
		name     string
		cfg      *Config
		explicit string
		wantName string
		wantIP   string
		wantErr  string // substring expected in the error, "" means no error
		wantNil  bool
	}{
		{
			name:     "explicit name found",
			cfg:      twoProfiles(),
			explicit: "work",
			wantName: "work",
			wantIP:   "10.0.0.1",
		},
		{
			name:     "explicit name missing lists available",
			cfg:      twoProfiles(),
			explicit: "nope",
			wantErr:  `unknown profile "nope" (available: home, work)`,
		},
		{
			name:    "zero profiles returns nil",
			cfg:     &Config{Profiles: map[string]Profile{}, NetworkCache: map[string]string{}},
			wantNil: true,
		},
		{
			name:     "single profile without default falls back",
			cfg:      &Config{Profiles: map[string]Profile{"only": {IP: "192.168.0.9"}}, NetworkCache: map[string]string{}},
			wantName: "only",
			wantIP:   "192.168.0.9",
		},
		{
			name:    "multiple profiles without default errors",
			cfg:     &Config{Profiles: map[string]Profile{"a": {IP: "1.1.1.1"}, "b": {IP: "2.2.2.2"}}, NetworkCache: map[string]string{}},
			wantErr: "no default profile set",
		},
		{
			name:    "default profile missing errors",
			cfg:     &Config{DefaultProfile: "ghost", Profiles: map[string]Profile{"a": {IP: "1.1.1.1"}}, NetworkCache: map[string]string{}},
			wantErr: `default profile "ghost" not found (available: a)`,
		},
		{
			name:    "nil config",
			cfg:     nil,
			wantNil: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, name, err := ActiveProfile(tc.cfg, tc.explicit)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNil {
				if p != nil || name != "" {
					t.Fatalf("got (%+v, %q), want (nil, \"\")", p, name)
				}
				return
			}
			if name != tc.wantName || p == nil || p.IP != tc.wantIP {
				t.Fatalf("got (%+v, %q), want (name %q, IP %q)", p, name, tc.wantName, tc.wantIP)
			}
		})
	}
}

func TestConfigMalformedBacksUpAndReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	path, _ := configPath()
	if err := os.WriteFile(path, []byte("{not valid json"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error for malformed config, got %v", err)
	}
	if cfg == nil || len(cfg.Profiles) != 0 {
		t.Fatal("expected empty config for malformed file")
	}

	backups, err := filepath.Glob(path + ".bad-*")
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) == 0 {
		t.Fatal("expected a config.json.bad-* backup file")
	}
}

func TestKeyringKey(t *testing.T) {
	tests := []struct {
		profile string
		want    string
	}{
		{"", "password"},
		{"default", "password"},
		{"home", "password:home"},
	}
	for _, tc := range tests {
		if got := keyringKey(tc.profile); got != tc.want {
			t.Errorf("keyringKey(%q) = %q, want %q", tc.profile, got, tc.want)
		}
	}
}
