package main

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// MAC address patterns: aa:bb:cc:dd:ee:ff, aa-bb-cc-dd-ee-ff, aabb.ccdd.eeff, AABBCCDDEEFF
var macPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^([0-9a-fA-F]{2}[:-]){5}[0-9a-fA-F]{2}$`),   // aa:bb:cc:dd:ee:ff or aa-bb-cc-dd-ee-ff
	regexp.MustCompile(`^([0-9a-fA-F]{4}\.){2}[0-9a-fA-F]{4}$`),      // aabb.ccdd.eeff
	regexp.MustCompile(`^[0-9a-fA-F]{12}$`),                           // AABBCCDDEEFF
}

// ValidateMAC checks if the input is a valid MAC address.
func ValidateMAC(mac string) error {
	mac = strings.TrimSpace(mac)
	for _, re := range macPatterns {
		if re.MatchString(mac) {
			return nil
		}
	}
	return fmt.Errorf("invalid MAC address %q (expected format like aa:bb:cc:dd:ee:ff)", mac)
}

// NormalizeMAC converts any valid MAC format to lowercase colon-separated (aa:bb:cc:dd:ee:ff).
func NormalizeMAC(mac string) string {
	mac = strings.TrimSpace(mac)
	mac = strings.ToLower(mac)

	// Remove all separators
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")

	// Insert colons every 2 chars
	if len(mac) == 12 {
		return fmt.Sprintf("%s:%s:%s:%s:%s:%s",
			mac[0:2], mac[2:4], mac[4:6], mac[6:8], mac[8:10], mac[10:12])
	}
	return mac
}

// ValidateIPv4 checks if the input is a valid IPv4 address.
func ValidateIPv4(ip string) error {
	ip = strings.TrimSpace(ip)
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid IP address %q (expected format like 192.168.0.1)", ip)
	}
	if parsed.To4() == nil {
		return fmt.Errorf("invalid IPv4 address %q (IPv6 not supported)", ip)
	}
	// Reject IPv4-mapped IPv6 (::ffff:x.x.x.x)
	if strings.Contains(ip, ":") {
		return fmt.Errorf("invalid IPv4 address %q (IPv6 not supported)", ip)
	}
	return nil
}
