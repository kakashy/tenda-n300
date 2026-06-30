package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

func isTendaRouter(ip string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/login.html", ip))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(body)), "tenda")
}

func defaultGatewayIP() (string, error) {
	interfaces := []net.IP{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
				continue
			}
			interfaces = append(interfaces, ipnet.IP)
		}
	}
	if len(interfaces) == 0 {
		return "", fmt.Errorf("no network interfaces found")
	}
	ip := make(net.IP, 4)
	copy(ip, interfaces[0].To4())
	ip[3] = 1
	return ip.String(), nil
}

func parseHexIP(hex string) net.IP {
	if len(hex) < 8 {
		return nil
	}
	b := make([]byte, 4)
	for i := 0; i < 4; i++ {
		// Little-endian hex: bytes reversed
		val := hexToByte(hex[2*i])<<4 | hexToByte(hex[2*i+1])
		b[3-i] = val
	}
	return net.IPv4(b[0], b[1], b[2], b[3])
}

func hexToByte(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0
	}
}

func discoverRouters() ([]string, error) {
	var found []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	sema := make(chan struct{}, 50)

	gw, err := gatewayFromRoute()
	if err != nil {
		gw, err = defaultGatewayIP()
		if err != nil {
			gw = "192.168.0.1"
		}
	}

	if isTendaRouter(gw) {
		found = append(found, gw)
	}

	for _, ip := range generateCandidateIPs(gw) {
		wg.Add(1)
		sema <- struct{}{}
		ip := ip
		go func() {
			defer wg.Done()
			defer func() { <-sema }()
			if isTendaRouter(ip) {
				mu.Lock()
				found = append(found, ip)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return found, nil
}

func generateCandidateIPs(gw string) []string {
	ip := net.ParseIP(gw)
	if ip == nil {
		return nil
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return nil
	}
	network := uint32(ip4[0])<<24 | uint32(ip4[1])<<16 | uint32(ip4[2])<<8
	var candidates []string
	for i := 1; i <= 254; i++ {
		ip := fmt.Sprintf("%d.%d.%d.%d", byte(network>>24), byte(network>>16), byte(network>>8), byte(i))
		if ip == gw {
			continue
		}
		candidates = append(candidates, ip)
	}
	return candidates
}
