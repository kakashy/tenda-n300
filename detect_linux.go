//go:build linux

package main

import (
	"bufio"
	"net"
	"os"
	"strings"
	"time"
)

func networkFingerprintOS() (NetworkFingerprint, error) {
	gw, err := gatewayFromRoute()
	if err != nil {
		return NetworkFingerprint{}, err
	}
	fp := NetworkFingerprint{Gateway: gw}
	mac := arpLookupLinux(gw)
	if mac == "" {
		if conn, err := net.DialTimeout("tcp", gw+":80", 200*time.Millisecond); err == nil {
			conn.Close()
			mac = arpLookupLinux(gw)
		}
	}
	fp.GatewayMAC = mac
	return fp, nil
}

func arpLookupLinux(gw string) string {
	f, err := os.Open("/proc/net/arp")
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 || fields[0] != gw {
			continue
		}
		mac := fields[3]
		if mac == "" || mac == "00:00:00:00:00:00" {
			continue
		}
		return normalizeMAC(mac)
	}
	return ""
}
