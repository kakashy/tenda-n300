package main

import (
	"testing"
)

func TestValidateMAC(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"aa:bb:cc:dd:ee:ff", false},
		{"AA:BB:CC:DD:EE:FF", false},
		{"aa-bb-cc-dd-ee-ff", false},
		{"AA-BB-CC-DD-EE-FF", false},
		{"aabb.ccdd.eeff", false},
		{"AABB.CCDD.EEFF", false},
		{"AABBCCDDEEFF", false},
		{"aabbccddeeff", false},
		{"aa:bb-cc:dd:ee:ff", false}, // mixed separators
		{"", true},
		{"invalid", true},
		{"aa:bb:cc:dd:ee", true},       // too short
		{"aa:bb:cc:dd:ee:ff:00", true}, // too long
		{"gg:hh:ii:jj:kk:ll", true},    // invalid hex
		{"aa:bb:cc:dd:ee:ff:gg", true}, // invalid hex
		{"192.168.0.1", true},          // IP address
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateMAC(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMAC(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeMAC(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"aa:bb:cc:dd:ee:ff", "aa:bb:cc:dd:ee:ff"},
		{"AA:BB:CC:DD:EE:FF", "aa:bb:cc:dd:ee:ff"},
		{"aa-bb-cc-dd-ee-ff", "aa:bb:cc:dd:ee:ff"},
		{"AA-BB-CC-DD-EE-FF", "aa:bb:cc:dd:ee:ff"},
		{"aabb.ccdd.eeff", "aa:bb:cc:dd:ee:ff"},
		{"AABB.CCDD.EEFF", "aa:bb:cc:dd:ee:ff"},
		{"AABBCCDDEEFF", "aa:bb:cc:dd:ee:ff"},
		{"aabbccddeeff", "aa:bb:cc:dd:ee:ff"},
		{"aa:bb-cc:dd:ee:ff", "aa:bb:cc:dd:ee:ff"},
		{"  AA:BB:CC:DD:EE:FF  ", "aa:bb:cc:dd:ee:ff"}, // with whitespace
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeMAC(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeMAC(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateChannel(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"1", false},
		{"6", false},
		{"11", false},
		{" 3 ", false},
		{"0", true},
		{"12", true},
		{"-1", true},
		{"abc", true},
		{"", true},
		{"1.5", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateChannel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateChannel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateIPv4(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"192.168.0.1", false},
		{"10.0.0.1", false},
		{"255.255.255.255", false},
		{"0.0.0.0", false},
		{"192.168.0.1 ", false}, // with trailing space (trimmed)
		{" 192.168.0.1", false}, // with leading space (trimmed)
		{"", true},
		{"invalid", true},
		{"192.168.0", true},          // too few octets
		{"192.168.0.1.2", true},      // too many octets
		{"999.999.999.999", true},    // out of range
		{"192.168.001.001", true},    // leading zeros
		{"::ffff:192.168.0.1", true}, // IPv4-mapped IPv6
		{"::1", true},                // IPv6 loopback
		{"localhost", true},          // hostname
		{"192.168.0.1/24", true},     // CIDR notation
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateIPv4(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIPv4(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
