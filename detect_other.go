//go:build !linux && !darwin && !windows

package main

import "fmt"

func networkFingerprintOS() (NetworkFingerprint, error) {
	return NetworkFingerprint{}, fmt.Errorf("network fingerprint not implemented for this platform")
}
