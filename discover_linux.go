//go:build linux

package main

import (
	"fmt"
	"os"
	"strings"
)

func gatewayFromRoute() (string, error) {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if fields[1] == "00000000" {
			gw := parseHexIP(fields[2])
			if gw != nil {
				return gw.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no default gateway found")
}
