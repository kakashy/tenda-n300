package main

import "strings"

type NetworkFingerprint struct {
	Gateway    string
	GatewayMAC string
}

var networkFingerprint = defaultNetworkFingerprint

func defaultNetworkFingerprint() (NetworkFingerprint, error) {
	return networkFingerprintOS()
}

// fingerprintKey returns a stable key identifying a network by its default
// gateway IP and MAC. An empty gateway yields an empty key.
func fingerprintKey(fp NetworkFingerprint) string {
	if fp.Gateway == "" {
		return ""
	}
	return fp.Gateway + "|" + fp.GatewayMAC
}

// detectProfile resolves the profile to use for the current network. An
// explicit profile always wins. Otherwise the network fingerprint is looked up
// in the config's cache. If the fingerprint cannot be determined (e.g. offline)
// the default profile is returned as a best effort.
func detectProfile(cfg *Config, explicit string, fp NetworkFingerprint) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if cfg == nil {
		return "", nil
	}
	if fp.Gateway == "" {
		return cfg.DefaultProfile, nil
	}
	if name, ok := cfg.NetworkCache[fingerprintKey(fp)]; ok {
		return name, nil
	}
	return "", nil
}

// recordNetworkProfile writes the fingerprint->profile mapping into the
// network cache so the profile can be auto-detected on future runs. It
// returns false (and records nothing) when the fingerprint has no gateway or
// no gateway MAC, since an IP-only key would collide across networks sharing
// a gateway address.
func recordNetworkProfile(cfg *Config, fp NetworkFingerprint, profile string) bool {
	if cfg == nil {
		return false
	}
	key := fingerprintKey(fp)
	if key == "" || fp.GatewayMAC == "" {
		return false
	}
	if cfg.NetworkCache == nil {
		cfg.NetworkCache = map[string]string{}
	}
	cfg.NetworkCache[key] = profile
	trimNetworkCache(cfg.NetworkCache, key)
	return true
}

// trimNetworkCache keeps the cache at most maxNetworkCache entries, preferring
// to evict entries other than keepKey. It is used after recording a detected
// profile so a recently-learned mapping is never the first to go.
const maxNetworkCache = 10

func trimNetworkCache(cache map[string]string, keepKey string) {
	if cache == nil {
		return
	}
	for len(cache) > maxNetworkCache {
		evicted := false
		for k := range cache {
			if k != keepKey {
				delete(cache, k)
				evicted = true
				break
			}
		}
		if !evicted {
			// Nothing left to evict (every key is keepKey); bail out to avoid
			// an infinite loop.
			return
		}
	}
}

// normalizeMAC canonicalizes an ARP entry's MAC to lowercase colon-separated
// form (aa:bb:cc:dd:ee:ff). macOS `arp` prints octets without leading zeros
// (e.g. "0:11:22:33:44:55"), so those are zero-padded first.
func normalizeMAC(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.Contains(s, ":") {
		parts := strings.Split(s, ":")
		if len(parts) == 6 {
			var sb strings.Builder
			for i, p := range parts {
				p = strings.TrimSpace(p)
				if len(p) == 1 {
					p = "0" + p
				}
				if i > 0 {
					sb.WriteByte(':')
				}
				sb.WriteString(p)
			}
			s = sb.String()
		}
	}
	return NormalizeMAC(s)
}
