package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
)

const keyringService = "tenda-n300"

var profileNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,31}$`)

type Profile struct {
	IP string `json:"ip"`
}

type Config struct {
	DefaultProfile string             `json:"default_profile,omitempty"`
	Profiles       map[string]Profile `json:"profiles"`
	NetworkCache   map[string]string  `json:"network_cache,omitempty"` // fingerprint -> profile name
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "tenda-n300")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func emptyConfig() *Config {
	return &Config{
		Profiles:     map[string]Profile{},
		NetworkCache: map[string]string{},
	}
}

// backupBadConfig preserves an unparseable config file before it is replaced,
// so a malformed file never silently destroys the user's profiles.
func backupBadConfig(path string, data []byte) error {
	return os.WriteFile(fmt.Sprintf("%s.bad-%d", path, time.Now().Unix()), data, 0600)
}

func LoadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return emptyConfig(), nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return emptyConfig(), nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		backupBadConfig(path, data)
		return emptyConfig(), nil
	}

	// Legacy format: top-level "ip" with no "profiles" key.
	if _, hasIP := raw["ip"]; hasIP {
		if _, hasProfiles := raw["profiles"]; !hasProfiles {
			var legacy struct {
				IP string `json:"ip"`
			}
			if err := json.Unmarshal(data, &legacy); err != nil {
				backupBadConfig(path, data)
				return emptyConfig(), nil
			}
			cfg := &Config{
				DefaultProfile: "default",
				Profiles: map[string]Profile{
					"default": {IP: legacy.IP},
				},
				NetworkCache: map[string]string{},
			}
			if err := SaveConfig(cfg); err != nil {
				return nil, err
			}
			return cfg, nil
		}
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		backupBadConfig(path, data)
		return emptyConfig(), nil
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	if cfg.NetworkCache == nil {
		cfg.NetworkCache = map[string]string{}
	}
	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "config-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	if !jsonOutput {
		fmt.Fprintf(os.Stderr, "saved config to %s\n", path)
	}
	return nil
}

// ActiveProfile resolves the profile to use. An explicit name wins; otherwise
// the default profile is used, falling back to the sole profile when exactly
// one exists. With zero profiles it returns (nil, "", nil) meaning no profile
// is configured yet.
func ActiveProfile(cfg *Config, name string) (*Profile, string, error) {
	if cfg == nil {
		return nil, "", nil
	}
	if name != "" {
		p, ok := cfg.Profiles[name]
		if !ok {
			return nil, "", fmt.Errorf("unknown profile %q (available: %s)", name, profileNames(cfg))
		}
		return &p, name, nil
	}
	if cfg.DefaultProfile != "" {
		p, ok := cfg.Profiles[cfg.DefaultProfile]
		if !ok {
			return nil, "", fmt.Errorf("default profile %q not found (available: %s)", cfg.DefaultProfile, profileNames(cfg))
		}
		return &p, cfg.DefaultProfile, nil
	}
	switch len(cfg.Profiles) {
	case 0:
		return nil, "", nil
	case 1:
		for n, p := range cfg.Profiles {
			return &p, n, nil
		}
	}
	return nil, "", fmt.Errorf("no default profile set (use --profile or `profile use`)")
}

func profileNames(cfg *Config) string {
	names := make([]string, 0, len(cfg.Profiles))
	for n := range cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func validateProfileName(name string) error {
	if !profileNameRe.MatchString(name) {
		return fmt.Errorf("invalid profile name %q: must be 1-32 characters, start with a letter or digit, and contain only letters, digits, underscores, or hyphens (no colons)", name)
	}
	return nil
}

func keyringKey(profile string) string {
	if profile == "" || profile == "default" {
		return "password"
	}
	return "password:" + profile
}

func keyringGetPassword(profile string) (string, error) {
	return keyring.Get(keyringService, keyringKey(profile))
}

func keyringSetPassword(profile, pwd string) error {
	return keyring.Set(keyringService, keyringKey(profile), pwd)
}

func keyringDeletePassword(profile string) error {
	return keyring.Delete(keyringService, keyringKey(profile))
}

func keyringDeleteAllPasswords() error {
	return keyring.DeleteAll(keyringService)
}
