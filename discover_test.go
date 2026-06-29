package main

import (
	"net"
	"testing"
)

func TestHexToByte(t *testing.T) {
	tests := []struct {
		c    byte
		want byte
	}{
		{'0', 0}, {'1', 1}, {'5', 5}, {'9', 9},
		{'a', 10}, {'f', 15}, {'A', 10}, {'F', 15},
		{'z', 0}, {'g', 0},
	}
	for _, tc := range tests {
		got := hexToByte(tc.c)
		if got != tc.want {
			t.Errorf("hexToByte(%q) = %d, want %d", tc.c, got, tc.want)
		}
	}
}

func TestParseHexIP(t *testing.T) {
	tests := []struct {
		hex     string
		want    string
		wantNil bool
	}{
		{"0101A8C0", "192.168.1.1", false},  // C0A80101 in little-endian
		{"00000000", "0.0.0.0", false},
		{"FFFFFFFF", "255.255.255.255", false},
		{"6401A8C0", "192.168.1.100", false},
		{"", "", true},
		{"abcd", "", true},
	}
	for _, tc := range tests {
		got := parseHexIP(tc.hex)
		if tc.wantNil {
			if got != nil {
				t.Errorf("parseHexIP(%q) = %v, want nil", tc.hex, got)
			}
			continue
		}
		if got == nil {
			t.Errorf("parseHexIP(%q) = nil, want %s", tc.hex, tc.want)
			continue
		}
		if got.String() != tc.want {
			t.Errorf("parseHexIP(%q) = %s, want %s", tc.hex, got, tc.want)
		}
	}
}

func TestGenerateCandidateIPs(t *testing.T) {
	ips := generateCandidateIPs("192.168.1.1")
	if len(ips) != 253 {
		t.Fatalf("expected 253 candidates, got %d", len(ips))
	}
	seen := make(map[string]bool)
	for _, ip := range ips {
		if ip == "192.168.1.1" {
			t.Error("candidates should not include the gateway itself")
		}
		if seen[ip] {
			t.Errorf("duplicate IP: %s", ip)
		}
		seen[ip] = true
		if net.ParseIP(ip) == nil {
			t.Errorf("invalid IP: %s", ip)
		}
	}
}

func TestGenerateCandidateIPsInvalid(t *testing.T) {
	if got := generateCandidateIPs("not-an-ip"); got != nil {
		t.Fatalf("expected nil for invalid input, got %v", got)
	}
}

func TestGenerateCandidateIPsIPv6(t *testing.T) {
	if got := generateCandidateIPs("::1"); got != nil {
		t.Fatalf("expected nil for IPv6, got %v", got)
	}
}

func TestGenerateCandidateIPsOtherSubnet(t *testing.T) {
	ips := generateCandidateIPs("10.0.0.1")
	if len(ips) != 253 {
		t.Fatalf("expected 253 candidates, got %d", len(ips))
	}
	// IPs should be 10.0.0.x
	for _, ip := range ips {
		if len(ip) < 7 || ip[:6] != "10.0.0" {
			t.Errorf("unexpected IP in 10.0.0.x range: %s", ip)
		}
	}
}
