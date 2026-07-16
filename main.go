package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var version = "dev"

func main() {
	setupSignalHandler()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `tenda-n300 — control your Tenda N300 router

Usage:
	tenda-n300 [--ip <addr>] [--password <pass>] [--json] <command> [args]

Commands:
  devices               List connected devices
  block <mac> [mac2 ...]  Block one or more devices by MAC address
  unblock <mac> [mac2 ...]  Unblock one or more devices by MAC address
  firmwareinfo          Show router firmware information
  status                Show router summary
  wifi                  Show WiFi settings (SSID, password, channel, encryption)
  wifi --ssid <name> --wifi-password <pass> --channel <n> --encrypt <mode>
                        Change WiFi settings (any combination)

  reboot                Reboot the router
  reset                 Factory reset router (wipes all config)
  backup [file]         Download config backup
  restore <file>        Restore config from backup file
  syslog [file]         Export system log
  ping                  Check if router is reachable and responsive
  discover              Scan network for Tenda routers
  config                Show current config
  config set <key> <val>  Set config key (ip, password)
  uninstall             Remove binary, config, and stored credentials
  completion bash|zsh    Generate shell completion script
  version               Show version

Flags:
  --ip <addr>        Router IP address (overrides config)
  --password <pass>  Router admin password (overrides config)
  --json             Output as JSON (machine-readable)
  --ssid <name>           New WiFi SSID (for wifi command)
  --wifi-password <pass>  New WiFi password (for wifi command)
  --channel <n>           New WiFi channel (1-11) (for wifi command)
  --encrypt <mode>        New WiFi encryption mode (for wifi command)
`)
	}

	var ip string
	var password string
	showVersion := flag.Bool("version", false, "show version")
	flag.BoolVar(&jsonOutput, "json", false, "output as JSON")
	flag.StringVar(&ip, "ip", "", "router IP address")
	flag.StringVar(&password, "password", "", "router admin password")
	flag.Parse()

	if *showVersion {
		if jsonOutput {
			printJSON(map[string]string{"version": version})
		} else {
			fmt.Println("tenda-n300", version)
		}
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}
	if ip != "" {
		if err := ValidateIPv4(ip); err != nil {
			printError("%v", err)
			os.Exit(1)
		}
	}

	switch args[0] {
	case "version":
		if jsonOutput {
			printJSON(map[string]string{"version": version})
		} else {
			fmt.Println("tenda-n300", version)
		}
	case "discover":
		cmdDiscover()
	case "ping":
		cmdPing(ip)
	case "config":
		cmdConfig(args[1:])
	case "completion":
		cmdCompletion(args[1:])
	case "uninstall":
		cmdUninstall()
	case "devices", "status", "firmwareinfo", "wifi", "block", "unblock", "reboot", "reset", "backup", "restore", "syslog":
		for _, a := range args[1:] {
			if a == "--help" || a == "-h" {
				printSubcommandHelp(args[0])
				os.Exit(0)
			}
		}
		client := connectRouter(ip, password)
		switch args[0] {
		case "wifi":
			wifiFlags := flag.NewFlagSet("wifi", flag.ExitOnError)
			wifiSSID := wifiFlags.String("ssid", "", "new WiFi SSID")
			wifiPassword := wifiFlags.String("wifi-password", "", "new WiFi password")
			wifiChannel := wifiFlags.String("channel", "", "new WiFi channel (1-11)")
			wifiEncrypt := wifiFlags.String("encrypt", "", "new WiFi encryption mode")
			wifiFlags.Parse(args[1:])

			hasChanges := *wifiSSID != "" || *wifiPassword != "" || *wifiChannel != "" || *wifiEncrypt != ""

			if !hasChanges {
				startSpinner("fetching WiFi settings")
				settings, err := client.GetWiFiSettings()
				stopSpinner()
				if err != nil {
					printError("%v", err)
					os.Exit(1)
				}
				printWiFiSettings(settings)
			} else {
				startSpinner("fetching current WiFi settings")
				current, err := client.GetWiFiSettings()
				stopSpinner()
				if err != nil {
					printError("%v", err)
					os.Exit(1)
				}
				if *wifiSSID != "" {
					current.SSID = *wifiSSID
				}
				if *wifiPassword != "" {
					current.Password = *wifiPassword
				}
				if *wifiChannel != "" {
					current.Channel = *wifiChannel
				}
				if *wifiEncrypt != "" {
					current.Encryption = *wifiEncrypt
				}
				startSpinner("updating WiFi settings")
				err = client.SetWiFiSettings(current)
				stopSpinner()
				if err != nil {
					printError("%v", err)
					os.Exit(1)
				}
				if jsonOutput {
					printJSON(map[string]string{"status": "ok"})
				} else {
					fmt.Println("WiFi settings updated")
				}
			}
		case "devices":
			startSpinner("fetching devices")
			devices, err := client.GetDevices()
			stopSpinner()
			if err != nil {
				printError("%v", err)
				os.Exit(1)
			}
			printDeviceTable(devices)
		case "firmwareinfo":
			startSpinner("fetching firmware info")
			info, err := client.GetFirmwareInfo()
			stopSpinner()
			if err != nil {
				printError("%v", err)
				os.Exit(1)
			}
			printFirmwareInfo(info)
		case "status":
			startSpinner("fetching devices")
			devices, err := client.GetDevices()
			stopSpinner()
			if err != nil {
				printError("%v", err)
				os.Exit(1)
			}
			printStatus(devices)
		case "block":
			if len(args) < 2 {
				if jsonOutput {
					printJSON(map[string]string{"error": "usage: tenda-n300 block <mac> [mac2 mac3 ...]"})
				} else {
					fmt.Fprintln(os.Stderr, "usage: tenda-n300 block <mac> [mac2 mac3 ...]")
				}
				os.Exit(1)
			}
			macs := args[1:]
			var blockFailed bool
			if jsonOutput {
				var results []map[string]string
				for _, mac := range macs {
					if err := client.BlockMAC(mac); err != nil {
						blockFailed = true
						results = append(results, map[string]string{"status": "error", "mac": mac, "error": err.Error()})
					} else {
						results = append(results, map[string]string{"status": "ok", "mac": mac, "action": "block"})
					}
				}
				printJSON(results)
			} else {
				for _, mac := range macs {
					if err := client.BlockMAC(mac); err != nil {
						blockFailed = true
						fmt.Fprintf(os.Stderr, "error blocking %s: %v\n", mac, err)
					} else {
						fmt.Printf("blocked %s\n", mac)
					}
				}
			}
			if blockFailed {
				os.Exit(1)
			}
		case "unblock":
			if len(args) < 2 {
				if jsonOutput {
					printJSON(map[string]string{"error": "usage: tenda-n300 unblock <mac> [mac2 mac3 ...]"})
				} else {
					fmt.Fprintln(os.Stderr, "usage: tenda-n300 unblock <mac> [mac2 mac3 ...]")
				}
				os.Exit(1)
			}
			macs := args[1:]
			var unblockFailed bool
			if jsonOutput {
				var results []map[string]string
				for _, mac := range macs {
					if err := client.UnblockMAC(mac); err != nil {
						unblockFailed = true
						results = append(results, map[string]string{"status": "error", "mac": mac, "error": err.Error()})
					} else {
						results = append(results, map[string]string{"status": "ok", "mac": mac, "action": "unblock"})
					}
				}
				printJSON(results)
			} else {
				for _, mac := range macs {
					if err := client.UnblockMAC(mac); err != nil {
						unblockFailed = true
						fmt.Fprintf(os.Stderr, "error unblocking %s: %v\n", mac, err)
					} else {
						fmt.Printf("unblocked %s\n", mac)
					}
				}
			}
			if unblockFailed {
				os.Exit(1)
			}

		case "reboot":
			if err := client.Reboot(); err != nil {
				printError("%v", err)
				os.Exit(1)
			}
			if jsonOutput {
				printJSON(map[string]string{"status": "ok"})
			} else {
				fmt.Println("rebooting...")
			}

		case "reset":
			if !jsonOutput {
				fmt.Print("are you sure? this will wipe all config. type 'yes': ")
				var s string
				fmt.Scanln(&s)
				if s != "yes" {
					os.Exit(0)
				}
			}
			if err := client.Reset(); err != nil {
				printError("%v", err)
				os.Exit(1)
			}
			if jsonOutput {
				printJSON(map[string]string{"status": "ok"})
			} else {
				fmt.Println("resetting to factory defaults...")
			}

		case "backup":
			startSpinner("downloading config")
			data, err := client.BackupConfig()
			stopSpinner()
			if err != nil {
				printError("%v", err)
				os.Exit(1)
			}
			dest := "RouterCfm.cfg"
			if len(args) > 1 {
				dest = args[1]
			}
			if err := os.WriteFile(dest, data, 0644); err != nil {
				printError("%v", err)
				os.Exit(1)
			}
			if jsonOutput {
				printJSON(map[string]string{"status": "ok", "file": dest})
			} else {
				fmt.Printf("config saved to %s\n", dest)
			}

		case "restore":
			if len(args) < 2 {
				if jsonOutput {
					printJSON(map[string]string{"error": "usage: tenda-n300 restore <file>"})
				} else {
					fmt.Fprintln(os.Stderr, "usage: tenda-n300 restore <file>")
				}
				os.Exit(1)
			}
			if err := client.RestoreConfig(args[1]); err != nil {
				printError("%v", err)
				os.Exit(1)
			}
			if jsonOutput {
				printJSON(map[string]string{"status": "ok"})
			} else {
				fmt.Println("config restored, rebooting...")
			}

		case "syslog":
			startSpinner("downloading syslog")
			data, err := client.ExportSyslog()
			stopSpinner()
			if err != nil {
				printError("%v", err)
				os.Exit(1)
			}
			if len(args) > 1 {
				dest := args[1]
				if err := os.WriteFile(dest, data, 0644); err != nil {
					printError("%v", err)
					os.Exit(1)
				}
				if jsonOutput {
					printJSON(map[string]string{"status": "ok", "file": dest})
				} else {
					fmt.Printf("syslog saved to %s\n", dest)
				}
			} else {
				if jsonOutput {
					printJSON(map[string]any{"status": "ok", "data": string(data)})
				} else {
					os.Stdout.Write(data)
				}
			}
		}
	default:
		printError("unknown command: %s", args[0])
		if !jsonOutput {
			flag.Usage()
		}
		os.Exit(1)
	}
}

func connectRouter(ip, password string) *RouterClient {
	if ip == "" || password == "" {
		cfg, err := LoadConfig()
		if err != nil {
			printError("config error: %v", err)
			os.Exit(1)
		}
		if ip == "" {
			ip = cfg.IP
			if ip != "" {
				if err := ValidateIPv4(ip); err != nil {
					printError("invalid IP in config: %v", err)
					os.Exit(1)
				}
			}
		}
		if password == "" {
			password, err = keyringGetPassword()
			if err != nil {
				printError("no password set")
				if !jsonOutput {
					fmt.Fprintf(os.Stderr, "  set it:  tenda-n300 config set password <pass>\n  or run:   tenda-n300 --password <pass> <command>\n")
				}
				os.Exit(1)
			}
		}
	}
	if ip == "" {
		guess := "192.168.0.1"
		if jsonOutput {
			printError("no IP set")
			os.Exit(1)
		}
		fmt.Printf("Router IP not set. Try %s? [Y/n]: ", guess)
		var s string
		fmt.Scanln(&s)
		if s == "" || s == "y" || s == "Y" || s == "yes" {
			ip = guess
			// Offer to save it
			fmt.Printf("Save %s to config for next time? [y/N]: ", ip)
			var save string
			fmt.Scanln(&save)
			if save == "y" || save == "Y" || save == "yes" {
				cfg, _ := LoadConfig()
				if cfg == nil {
					cfg = &Config{}
				}
				cfg.IP = ip
				SaveConfig(cfg)
			}
		} else {
			fmt.Print("Enter router IP: ")
			fmt.Scanln(&ip)
			if ip == "" {
				printError("no IP provided")
				os.Exit(1)
			}
			if err := ValidateIPv4(ip); err != nil {
				printError("%v", err)
				os.Exit(1)
			}
		}
	}
	if password == "" {
		printError("router password required (use --password or `config set password`)")
		os.Exit(1)
	}
	client, err := NewRouterClient(ip, password)
	if err != nil {
		printError("login failed: %v", err)
		os.Exit(1)
	}
	return client
}

func cmdDiscover() {
	startSpinner("scanning network")
	routers, err := discoverRouters()
	stopSpinner()
	if err != nil {
		printError("discovery error: %v", err)
		os.Exit(1)
	}
	if jsonOutput {
		printJSON(routers)
		return
	}
	if len(routers) == 0 {
		fmt.Println("no Tenda routers found on the local network")
		return
	}
	fmt.Println("found Tenda router(s):")
	for _, r := range routers {
		fmt.Println(" ", r)
	}
}

func cmdPing(ip string) {
	if ip == "" {
		cfg, err := LoadConfig()
		if err != nil {
			printError("config error: %v", err)
			os.Exit(1)
		}
		ip = cfg.IP
	}
	if ip == "" {
		printError("no router IP set (use --ip or `config set ip`)")
		os.Exit(1)
	}
	if err := ValidateIPv4(ip); err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	startSpinner("pinging router")
	result := PingRouter(ip)
	stopSpinner()
	printPingResult(result)
}

func cmdUninstall() {
	if !jsonOutput {
		fmt.Print("remove the binary, config, and stored credentials? type 'yes': ")
		var s string
		fmt.Scanln(&s)
		if s != "yes" {
			os.Exit(0)
		}
	}

	var errs []string

	// Remove keyring entry
	if err := keyringDeletePassword(); err != nil {
		errs = append(errs, fmt.Sprintf("keyring: %v", err))
	}

	// Remove config directory
	configDir, err := configDir()
	if err != nil {
		errs = append(errs, fmt.Sprintf("config dir: %v", err))
	} else if configDir != "" {
		if err := os.RemoveAll(configDir); err != nil {
			errs = append(errs, fmt.Sprintf("config dir remove: %v", err))
		}
	}

	// Remove shell completions (paths mirror install.sh)
	home, _ := os.UserHomeDir()
	completionPaths := []string{
		"/etc/bash_completion.d/tenda-n300",
		"/usr/local/share/bash-completion/completions/tenda-n300",
		filepath.Join(home, ".local/share/bash-completion/completions/tenda-n300"),
		"/usr/local/share/zsh/site-functions/_tenda-n300",
		"/usr/share/zsh/site-functions/_tenda-n300",
		filepath.Join(home, ".zsh/completion/_tenda-n300"),
	}
	for _, p := range completionPaths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("completion %s: %v", p, err))
		}
	}

	// Remove the binary itself (works on Linux/macOS)
	bin, err := os.Executable()
	if err != nil {
		errs = append(errs, fmt.Sprintf("find binary: %v", err))
	} else if runtime.GOOS == "windows" {
		fmt.Fprintf(os.Stderr, "remove the binary manually: del %s\n", bin)
	} else if err := os.Remove(bin); err != nil {
		errs = append(errs, fmt.Sprintf("binary remove: %v", err))
	}

	if len(errs) > 0 {
		if jsonOutput {
			printJSON(map[string]any{"status": "error", "errors": errs})
		} else {
			fmt.Fprintln(os.Stderr, "uninstall completed with errors:")
			for _, e := range errs {
				fmt.Fprintln(os.Stderr, "  -", e)
			}
		}
		os.Exit(1)
	}

	if jsonOutput {
		printJSON(map[string]string{"status": "ok"})
	} else {
		fmt.Println("uninstalled")
	}
}

func cmdConfig(args []string) {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			fmt.Fprintf(os.Stderr, `Usage: tenda-n300 config [set <key> <value>]

Show or set configuration.

Subcommands:
  set <key> <value>  Set a config key (ip, password)

Keys:
  ip        Router IP address
  password  Router admin password (stored in OS keyring)
`)
			os.Exit(0)
		}
	}
	if len(args) == 0 {
		cfg, err := LoadConfig()
		if err != nil {
			printError("%v", err)
			os.Exit(1)
		}
		if jsonOutput {
			printJSON(cfg)
			return
		}
		fmt.Printf("ip:       %s\n", cfg.IP)
		if _, err := keyringGetPassword(); err == nil {
			fmt.Println("password: <set (stored in OS keyring)>")
		} else {
			fmt.Println("password: <not set>")
		}
		return
	}
	if len(args) < 2 {
		if jsonOutput {
			printJSON(map[string]string{"error": "usage: tenda-n300 config set <key> <value>"})
		} else {
			fmt.Fprintln(os.Stderr, "usage: tenda-n300 config set <key> <value>")
		}
		os.Exit(1)
	}
	switch args[0] {
	case "set":
		key := args[1]
		val := strings.Join(args[2:], " ")
		cfg, err := LoadConfig()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			printError("%v", err)
			os.Exit(1)
		}
		if cfg == nil {
			cfg = &Config{}
		}
		switch key {
		case "ip":
			if err := ValidateIPv4(val); err != nil {
				printError("%v", err)
				os.Exit(1)
			}
			cfg.IP = val
		case "password":
			if err := keyringSetPassword(val); err != nil {
				printError("failed to save password to OS keyring: %v", err)
				os.Exit(1)
			}
			if !jsonOutput {
				fmt.Fprintln(os.Stderr, "password saved to OS keyring")
			}
			return
		default:
			printError("unknown config key: %s (use ip or password)", key)
			os.Exit(1)
		}
		if err := SaveConfig(cfg); err != nil {
			printError("%v", err)
			os.Exit(1)
		}
	default:
		printError("unknown config subcommand: %s", args[0])
		os.Exit(1)
	}
}

func printSubcommandHelp(cmd string) {
	help := map[string]string{
		"devices":      "Usage: tenda-n300 devices\n\nList all connected devices with their MAC addresses, hostnames, and IPs.",
		"block":        "Usage: tenda-n300 block <mac> [mac2 ...]\n\nBlock one or more devices by MAC address.",
		"unblock":      "Usage: tenda-n300 unblock <mac> [mac2 ...]\n\nUnblock one or more devices by MAC address.",
		"firmwareinfo": "Usage: tenda-n300 firmwareinfo\n\nShow router firmware information.",
		"wifi":         "Usage: tenda-n300 wifi [--ssid <name>] [--wifi-password <pass>] [--channel <n>] [--encrypt <mode>]\n\nShow WiFi settings. Pass flags to change settings (any combination).",
		"status":       "Usage: tenda-n300 status\n\nShow router summary with connected devices.",
		"reboot":       "Usage: tenda-n300 reboot\n\nReboot the router.",
		"reset":        "Usage: tenda-n300 reset\n\nFactory reset router (wipes all config). Prompts for confirmation.",
		"backup":       "Usage: tenda-n300 backup [file]\n\nDownload config backup. Defaults to RouterCfm.cfg.",
		"restore":      "Usage: tenda-n300 restore <file>\n\nRestore config from a backup file.",
		"syslog":       "Usage: tenda-n300 syslog [file]\n\nExport system log. Writes to file if given, otherwise stdout.",
	}
	if h, ok := help[cmd]; ok {
		fmt.Fprintln(os.Stderr, h)
	} else {
		fmt.Fprintf(os.Stderr, "no help available for %s\n", cmd)
	}
}
