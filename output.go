package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"
)

var (
	jsonOutput bool
	spinnerStop chan struct{}
)

func startSpinner(msg string) {
	if jsonOutput {
		return
	}
	spinnerStop = make(chan struct{})
	go func() {
		chars := []string{"|", "/", "-", "\\"}
		i := 0
		for {
			select {
			case <-spinnerStop:
				return
			default:
				fmt.Fprintf(os.Stderr, "\r%s %s ", chars[i%len(chars)], msg)
				i++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func stopSpinner() {
	if jsonOutput || spinnerStop == nil {
		return
	}
	close(spinnerStop)
	fmt.Fprint(os.Stderr, "\r\033[K")
	spinnerStop = nil
}

func printTable(header []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	for i, h := range header {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, h)
	}
	fmt.Fprintln(w)
	for _, row := range rows {
		for i, col := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, col)
		}
		fmt.Fprintln(w)
	}
	w.Flush()
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func printFirmwareInfo(info *FirmwareInfo) {
	if jsonOutput {
		printJSON(info)
		return
	}
	fmt.Printf("Version:         %s\n", info.Version)
	fmt.Printf("Uptime:          %s\n", info.Uptime)
	fmt.Printf("Default DNS:     %s\n", info.DefaultDNS)
	fmt.Printf("Alt DNS:         %s\n", info.AltDNS)
	fmt.Printf("Connection Type: %s\n", info.ConnectionType)
	fmt.Printf("Gateway:         %s\n", info.Gateway)
}

func printDeviceTable(devices []Device) {
	if jsonOutput {
		printJSON(devices)
		return
	}
	if len(devices) == 0 {
		fmt.Println("no devices found")
		return
	}
	rows := make([][]string, 0, len(devices))
	for _, d := range devices {
		access := "blocked"
		if d.Access {
			access = "allowed"
		}
		rows = append(rows, []string{d.Hostname, d.IP, d.MAC, d.Type, access})
	}
	printTable([]string{"HOSTNAME", "IP", "MAC", "TYPE", "ACCESS"}, rows)
}

func printStatus(devices []Device) {
	if jsonOutput {
		printJSON(map[string]any{
			"total":   len(devices),
			"online":  countAccess(devices, true),
			"blocked": countAccess(devices, false),
			"devices": devices,
		})
		return
	}
	fmt.Printf("total devices: %d\n", len(devices))
	fmt.Printf("online:        %d\n", countAccess(devices, true))
	fmt.Printf("blocked:       %d\n", countAccess(devices, false))
}

func countAccess(devices []Device, allowed bool) int {
	n := 0
	for _, d := range devices {
		if d.Access == allowed {
			n++
		}
	}
	return n
}
