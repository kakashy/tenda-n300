//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func gatewayFromRoute() (string, error) {
	out, err := exec.Command("route", "-n", "get", "default").Output()
	if err != nil {
		return "", fmt.Errorf("route command failed: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "gateway:") {
			gw := strings.TrimSpace(strings.TrimPrefix(line, "gateway:"))
			if gw != "" {
				return gw, nil
			}
		}
	}
	return "", fmt.Errorf("no default gateway found")
}
