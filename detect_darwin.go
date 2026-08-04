//go:build darwin

package main

import (
	"os/exec"
	"strings"
)

func networkFingerprintOS() (NetworkFingerprint, error) {
	gw, err := gatewayFromRoute()
	if err != nil {
		return NetworkFingerprint{}, err
	}
	fp := NetworkFingerprint{Gateway: gw}
	if out, err := exec.Command("arp", "-n", gw).Output(); err == nil {
		fp.GatewayMAC = arpLookupDarwin(string(out))
	}
	return fp, nil
}

func arpLookupDarwin(out string) string {
	for _, line := range strings.Split(out, "\n") {
		idx := strings.Index(line, " at ")
		if idx < 0 {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(line[idx+len(" at "):]))
		if len(fields) == 0 {
			continue
		}
		mac := fields[0]
		if mac == "" || mac == "(incomplete)" {
			continue
		}
		return normalizeMAC(mac)
	}
	return ""
}
