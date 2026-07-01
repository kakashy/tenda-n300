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

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `tenda-n300 — control your Tenda N300 router

Usage:
  tenda-n300 [--ip <addr>] [--password <pass>] [--json] <command> [args]

Commands:
  devices               List connected devices
  block <mac>           Block a device by MAC address
  unblock <mac>         Unblock a device by MAC address
  status                Show router summary
  reboot                Reboot the router
  reset                 Factory reset router (wipes all config)
  backup [file]         Download config backup
  restore <file>        Restore config from backup file
  syslog [file]         Export system log
  discover              Scan network for Tenda routers
  config                Show current config
  config set <key> <val>  Set config key (ip, password)
  uninstall             Remove binary, config, and stored credentials
  completion bash|zsh    Generate shell completion script

Flags:
  --ip <addr>        Router IP address (overrides config)
  --password <pass>  Router admin password (overrides config)
  --json             Output as JSON (machine-readable)
`)
	}

	var ip string
	var password string
	flag.BoolVar(&jsonOutput, "json", false, "output as JSON")
	flag.StringVar(&ip, "ip", "", "router IP address")
	flag.StringVar(&password, "password", "", "router admin password")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}
	if ip != "" {
		if err := ValidateIPv4(ip); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}

	switch args[0] {
	case "discover":
		cmdDiscover()
	case "config":
		cmdConfig(args[1:])
	case "completion":
		cmdCompletion(args[1:])
	case "uninstall":
		cmdUninstall()
	case "devices", "status", "block", "unblock", "reboot", "reset", "backup", "restore", "syslog":
		client := connectRouter(ip, password)
		switch args[0] {
		case "devices":
			startSpinner("fetching devices")
			devices, err := client.GetDevices()
			stopSpinner()
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			printDeviceTable(devices)
		case "status":
			startSpinner("fetching devices")
			devices, err := client.GetDevices()
			stopSpinner()
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			printStatus(devices)
		case "block":
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "usage: tenda-n300 block <mac>")
				os.Exit(1)
			}
			mac := args[1]
			if err := client.BlockMAC(mac); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			if jsonOutput {
				printJSON(map[string]string{"status": "ok", "mac": mac, "action": "block"})
			} else {
				fmt.Printf("blocked %s\n", mac)
			}
		case "unblock":
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "usage: tenda-n300 unblock <mac>")
				os.Exit(1)
			}
			mac := args[1]
			if err := client.UnblockMAC(mac); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			if jsonOutput {
				printJSON(map[string]string{"status": "ok", "mac": mac, "action": "unblock"})
			} else {
				fmt.Printf("unblocked %s\n", mac)
			}

		case "reboot":
			if err := client.Reboot(); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			fmt.Println("rebooting...")

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
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			fmt.Println("resetting to factory defaults...")

		case "backup":
			startSpinner("downloading config")
			data, err := client.BackupConfig()
			stopSpinner()
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			dest := "RouterCfm.cfg"
			if len(args) > 1 {
				dest = args[1]
			}
			if err := os.WriteFile(dest, data, 0644); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			fmt.Printf("config saved to %s\n", dest)

		case "restore":
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "usage: tenda-n300 restore <file>")
				os.Exit(1)
			}
			if err := client.RestoreConfig(args[1]); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			fmt.Println("config restored, rebooting...")

		case "syslog":
			startSpinner("downloading syslog")
			data, err := client.ExportSyslog()
			stopSpinner()
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			if len(args) > 1 {
				dest := args[1]
				if err := os.WriteFile(dest, data, 0644); err != nil {
					fmt.Fprintln(os.Stderr, "error:", err)
					os.Exit(1)
				}
				fmt.Printf("syslog saved to %s\n", dest)
			} else {
				os.Stdout.Write(data)
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		flag.Usage()
		os.Exit(1)
	}
}

func connectRouter(ip, password string) *RouterClient {
	if ip == "" || password == "" {
		cfg, err := LoadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "config error:", err)
			os.Exit(1)
		}
		if ip == "" {
			ip = cfg.IP
			if ip != "" {
				if err := ValidateIPv4(ip); err != nil {
					fmt.Fprintln(os.Stderr, "error: invalid IP in config:", err)
					os.Exit(1)
				}
			}
		}
		if password == "" {
			password, err = keyringGetPassword()
			if err != nil {
				fmt.Fprintf(os.Stderr, "config error: no password set\n  set it:  tenda-n300 config set password <pass>\n  or run:   tenda-n300 --password <pass> <command>\n")
				os.Exit(1)
			}
		}
	}
	if ip == "" {
		guess := "192.168.0.1"
		if jsonOutput {
			fmt.Fprintf(os.Stderr, "config error: no IP set\n")
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
				fmt.Fprintln(os.Stderr, "no IP provided")
				os.Exit(1)
			}
			if err := ValidateIPv4(ip); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
		}
	}
	if password == "" {
		fmt.Fprintln(os.Stderr, "error: router password required (use --password or `config set password`)")
		os.Exit(1)
	}
	client, err := NewRouterClient(ip, password)
	if err != nil {
		fmt.Fprintln(os.Stderr, "login failed:", err)
		os.Exit(1)
	}
	return client
}

func cmdDiscover() {
	startSpinner("scanning network")
	routers, err := discoverRouters()
	stopSpinner()
	if err != nil {
		fmt.Fprintln(os.Stderr, "discovery error:", err)
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
		fmt.Fprintln(os.Stderr, "uninstall completed with errors:")
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "  -", e)
		}
		os.Exit(1)
	}

	fmt.Println("uninstalled")
}

func cmdConfig(args []string) {
	if len(args) == 0 {
		cfg, err := LoadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
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
		fmt.Fprintln(os.Stderr, "usage: tenda-n300 config set <key> <value>")
		os.Exit(1)
	}
	switch args[0] {
	case "set":
		key := args[1]
		val := strings.Join(args[2:], " ")
		cfg, err := LoadConfig()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		if cfg == nil {
			cfg = &Config{}
		}
		switch key {
		case "ip":
			if err := ValidateIPv4(val); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			cfg.IP = val
		case "password":
			if err := keyringSetPassword(val); err != nil {
				fmt.Fprintln(os.Stderr, "error: failed to save password to OS keyring:", err)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "password saved to OS keyring")
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown config key: %s (use ip or password)\n", key)
			os.Exit(1)
		}
		if err := SaveConfig(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown config subcommand: %s\n", args[0])
		os.Exit(1)
	}
}
