package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"text/tabwriter"
	"time"
)

var (
	jsonOutput  bool
	spinnerMu   sync.Mutex
	spinnerStop chan struct{}
	exitFunc    func(int) = os.Exit
)

func setupSignalHandler() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go handleSignals(sigCh)
}

func handleSignals(sigCh <-chan os.Signal) {
	<-sigCh
	stopSpinner()
	exitFunc(130)
}

func startSpinner(msg string) {
	if jsonOutput {
		return
	}
	spinnerMu.Lock()
	if spinnerStop != nil {
		close(spinnerStop)
	}
	spinnerStop = make(chan struct{})
	ch := spinnerStop
	spinnerMu.Unlock()

	go func() {
		chars := []string{"|", "/", "-", "\\"}
		i := 0
		for {
			select {
			case <-ch:
				return
			default:
				fmt.Fprintf(os.Stderr, "\r%s %s ", chars[i%len(chars)], msg)
				i++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func spinnerActive() bool {
	spinnerMu.Lock()
	active := spinnerStop != nil
	spinnerMu.Unlock()
	return active
}

func stopSpinner() {
	if jsonOutput {
		return
	}
	spinnerMu.Lock()
	if spinnerStop == nil {
		spinnerMu.Unlock()
		return
	}
	close(spinnerStop)
	spinnerStop = nil
	spinnerMu.Unlock()
	fmt.Fprint(os.Stderr, "\r\033[K")
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

func printError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if jsonOutput {
		printJSON(map[string]string{"error": msg})
	} else {
		fmt.Fprintln(os.Stderr, "error:", msg)
	}
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
	fmt.Printf("WAN IP:          %s\n", info.WanIP)
	fmt.Printf("WAN MAC:         %s\n", info.WanMAC)
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

func printWiFiSettings(s *WiFiSettings) {
	if jsonOutput {
		printJSON(s)
		return
	}
	fmt.Printf("SSID:       %s\n", s.SSID)
	fmt.Printf("Password:   %s\n", s.Password)
	fmt.Printf("Channel:    %s\n", s.Channel)
	fmt.Printf("Encryption: %s\n", s.Encryption)
	fmt.Printf("Band:       %s\n", s.Band)
	fmt.Printf("WPS:        %s\n", s.WPS)
	fmt.Printf("Broadcast:  %s\n", s.Broadcast)
}

func printPingResult(r *PingResult) {
	if jsonOutput {
		printJSON(map[string]any{
			"reachable": r.Reachable,
			"latency_ms": r.Latency.Milliseconds(),
			"api_access": r.APIAccess,
			"router_ip":  r.RouterIP,
			"error":      r.Error,
		})
		return
	}
	if !r.Reachable {
		fmt.Printf("router %s is not reachable\n", r.RouterIP)
		if r.Error != "" {
			fmt.Printf("error: %s\n", r.Error)
		}
		return
	}
	api := "yes"
	if !r.APIAccess {
		api = "no"
	}
	fmt.Printf("router %s is reachable\n", r.RouterIP)
	fmt.Printf("latency:   %s\n", r.Latency.Round(time.Millisecond))
	fmt.Printf("api:       %s\n", api)
}
