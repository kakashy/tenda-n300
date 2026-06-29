package main

import (
	"os"
	"path/filepath"
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

	cfg.IP = "192.168.1.1"
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.IP != "192.168.1.1" {
		t.Fatalf("expected 192.168.1.1, got %s", loaded.IP)
	}
}

func TestConfigFilePermissions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := &Config{IP: "10.0.0.1"}
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
	if cfg.IP != "" {
		t.Fatal("expected empty config for missing file")
	}
}
