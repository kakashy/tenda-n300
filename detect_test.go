package main

import (
	"fmt"
	"testing"
)

func TestFingerprintKey(t *testing.T) {
	tests := []struct {
		name string
		fp   NetworkFingerprint
		want string
	}{
		{"full", NetworkFingerprint{Gateway: "192.168.0.1", GatewayMAC: "aa:bb:cc:dd:ee:ff"}, "192.168.0.1|aa:bb:cc:dd:ee:ff"},
		{"empty gateway", NetworkFingerprint{Gateway: "", GatewayMAC: "x"}, ""},
		{"empty mac", NetworkFingerprint{Gateway: "192.168.0.1", GatewayMAC: ""}, "192.168.0.1|"},
	}
	for _, tc := range tests {
		if got := fingerprintKey(tc.fp); got != tc.want {
			t.Errorf("%s: fingerprintKey(%+v) = %q, want %q", tc.name, tc.fp, got, tc.want)
		}
	}
}

func TestDetectProfile(t *testing.T) {
	cfg := &Config{
		DefaultProfile: "home",
		Profiles: map[string]Profile{
			"home": {IP: "192.168.0.1"},
			"work": {IP: "10.0.0.1"},
		},
		NetworkCache: map[string]string{
			"192.168.0.1|aa:bb:cc:dd:ee:ff": "home",
		},
	}

	t.Run("explicit wins", func(t *testing.T) {
		got, err := detectProfile(cfg, "work", NetworkFingerprint{})
		if err != nil {
			t.Fatal(err)
		}
		if got != "work" {
			t.Fatalf("got %q, want work", got)
		}
	})

	t.Run("cache hit", func(t *testing.T) {
		fp := NetworkFingerprint{Gateway: "192.168.0.1", GatewayMAC: "aa:bb:cc:dd:ee:ff"}
		got, err := detectProfile(cfg, "", fp)
		if err != nil {
			t.Fatal(err)
		}
		if got != "home" {
			t.Fatalf("got %q, want home", got)
		}
	})

	t.Run("cache miss", func(t *testing.T) {
		fp := NetworkFingerprint{Gateway: "10.0.0.1", GatewayMAC: "00:00:00:00:00:00"}
		got, err := detectProfile(cfg, "", fp)
		if err != nil {
			t.Fatal(err)
		}
		if got != "" {
			t.Fatalf("got %q, want empty string", got)
		}
	})

	t.Run("fingerprint error returns default", func(t *testing.T) {
		got, err := detectProfile(cfg, "", NetworkFingerprint{})
		if err != nil {
			t.Fatal(err)
		}
		if got != "home" {
			t.Fatalf("got %q, want home (DefaultProfile)", got)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		got, err := detectProfile(nil, "", NetworkFingerprint{})
		if err != nil {
			t.Fatal(err)
		}
		if got != "" {
			t.Fatalf("got %q, want empty string", got)
		}
	})

	t.Run("nil config with explicit name still wins", func(t *testing.T) {
		got, err := detectProfile(nil, "work", NetworkFingerprint{})
		if err != nil {
			t.Fatal(err)
		}
		if got != "work" {
			t.Fatalf("got %q, want work", got)
		}
	})
}

func TestRecordNetworkProfile(t *testing.T) {
	t.Run("records valid fingerprint", func(t *testing.T) {
		cfg := &Config{Profiles: map[string]Profile{}, NetworkCache: map[string]string{}}
		fp := NetworkFingerprint{Gateway: "192.168.0.1", GatewayMAC: "aa:bb:cc:dd:ee:ff"}
		if !recordNetworkProfile(cfg, fp, "home") {
			t.Fatal("expected record to succeed")
		}
		if cfg.NetworkCache["192.168.0.1|aa:bb:cc:dd:ee:ff"] != "home" {
			t.Fatalf("expected cached profile home, got %v", cfg.NetworkCache)
		}
	})

	t.Run("empty gateway not recorded", func(t *testing.T) {
		cfg := &Config{Profiles: map[string]Profile{}, NetworkCache: map[string]string{}}
		if recordNetworkProfile(cfg, NetworkFingerprint{}, "home") {
			t.Fatal("expected no record for empty gateway")
		}
		if len(cfg.NetworkCache) != 0 {
			t.Fatalf("expected empty cache, got %v", cfg.NetworkCache)
		}
	})

	t.Run("empty mac not recorded", func(t *testing.T) {
		cfg := &Config{Profiles: map[string]Profile{}, NetworkCache: map[string]string{}}
		fp := NetworkFingerprint{Gateway: "192.168.0.1", GatewayMAC: ""}
		if recordNetworkProfile(cfg, fp, "home") {
			t.Fatal("expected no record for empty gateway MAC")
		}
		if len(cfg.NetworkCache) != 0 {
			t.Fatalf("expected empty cache, got %v", cfg.NetworkCache)
		}
	})

	t.Run("nil config returns false", func(t *testing.T) {
		if recordNetworkProfile(nil, NetworkFingerprint{Gateway: "192.168.0.1", GatewayMAC: "aa:bb:cc:dd:ee:ff"}, "home") {
			t.Fatal("expected false for nil config")
		}
	})

	t.Run("trims to max keeping new entry", func(t *testing.T) {
		cfg := &Config{Profiles: map[string]Profile{}, NetworkCache: map[string]string{}}
		for i := 0; i < maxNetworkCache+2; i++ {
			cfg.NetworkCache[fmt.Sprintf("10.0.0.%d|mac-%02d", i, i)] = "p"
		}
		fp := NetworkFingerprint{Gateway: "192.168.0.1", GatewayMAC: "aa:bb:cc:dd:ee:ff"}
		recordNetworkProfile(cfg, fp, "home")
		if len(cfg.NetworkCache) != maxNetworkCache {
			t.Fatalf("expected %d entries, got %d", maxNetworkCache, len(cfg.NetworkCache))
		}
		if cfg.NetworkCache["192.168.0.1|aa:bb:cc:dd:ee:ff"] != "home" {
			t.Fatal("newly recorded entry was evicted")
		}
	})
}

func TestTrimNetworkCache(t *testing.T) {
	nEntries := func(n int) map[string]string {
		m := make(map[string]string, n)
		for i := 0; i < n; i++ {
			m[fmt.Sprintf("10.0.0.%d|mac-%02d", i, i)] = "p"
		}
		return m
	}

	t.Run("evicts down to cap keeping keepKey", func(t *testing.T) {
		cache := nEntries(12)
		cache["192.168.0.1|aa:bb:cc:dd:ee:ff"] = "home"
		trimNetworkCache(cache, "192.168.0.1|aa:bb:cc:dd:ee:ff")
		if len(cache) != maxNetworkCache {
			t.Fatalf("expected %d entries, got %d", maxNetworkCache, len(cache))
		}
		if cache["192.168.0.1|aa:bb:cc:dd:ee:ff"] != "home" {
			t.Fatal("keepKey entry was evicted")
		}
	})

	t.Run("at cap is left untouched", func(t *testing.T) {
		cache := nEntries(maxNetworkCache)
		trimNetworkCache(cache, "")
		if len(cache) != maxNetworkCache {
			t.Fatalf("expected %d entries, got %d", maxNetworkCache, len(cache))
		}
	})

	t.Run("below cap is left untouched", func(t *testing.T) {
		cache := nEntries(3)
		trimNetworkCache(cache, "nope")
		if len(cache) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(cache))
		}
	})

	t.Run("nil cache does not panic", func(t *testing.T) {
		trimNetworkCache(nil, "k")
	})

	t.Run("all entries are keepKey bails out", func(t *testing.T) {
		cache := map[string]string{
			"same|1": "a",
			"same|2": "b",
		}
		trimNetworkCache(cache, "same|1")
		if len(cache) != 2 {
			t.Fatalf("expected 2 entries kept, got %d", len(cache))
		}
	})

	t.Run("keepKey not present still trims", func(t *testing.T) {
		cache := nEntries(12)
		trimNetworkCache(cache, "ghost")
		if len(cache) != maxNetworkCache {
			t.Fatalf("expected %d entries, got %d", maxNetworkCache, len(cache))
		}
		if _, ok := cache["ghost"]; ok {
			t.Fatal("ghost should not be present")
		}
	})
}
