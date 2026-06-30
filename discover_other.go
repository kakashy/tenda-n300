//go:build !linux && !darwin && !windows

package main

import "fmt"

func gatewayFromRoute() (string, error) {
	return "", fmt.Errorf("gateway detection not implemented for this platform")
}
