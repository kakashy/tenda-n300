//go:build windows

package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func gatewayFromRoute() (string, error) {
	buflen := uint32(15000)
	buf := make([]byte, buflen)

	for {
		addr := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
		err := windows.GetAdaptersAddresses(
			windows.AF_INET,
			windows.GAA_FLAG_INCLUDE_GATEWAYS,
			0, addr, &buflen,
		)
		if err == nil {
			break
		}
		if err != windows.ERROR_BUFFER_OVERFLOW {
			return "", err
		}
		buf = make([]byte, buflen)
	}

	addr := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
	for ; addr != nil; addr = addr.Next {
		if addr.OperStatus != windows.IfOperStatusUp {
			continue
		}
		if addr.FirstGatewayAddress == nil {
			continue
		}
		ip := addr.FirstGatewayAddress.Address.IP()
		if ip != nil && ip.To4() != nil {
			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("no default gateway found")
}
