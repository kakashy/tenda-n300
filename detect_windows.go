//go:build windows

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
	if out, err := exec.Command("arp", "-a").Output(); err == nil {
		fp.GatewayMAC = arpLookupWindows(string(out), gw)
	}
	return fp, nil
}

func arpLookupWindows(out, gw string) string {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != gw {
			continue
		}
		return normalizeMAC(fields[1])
	}
	return ""
}
