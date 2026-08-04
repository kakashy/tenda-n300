package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

var version = "dev"

var profileFlag string

func main() {
	setupSignalHandler()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `tenda-n300 — control your Tenda N300 router

Usage:
	tenda-n300 [--ip <addr>] [--password <pass>] [--profile <name>] [--json] <command> [args]

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
  profile               Show active profile and list profiles
  profile add <name> [--ip <addr>] [--password <pass>]  Add a router profile
  profile set <name> [--ip <addr>] [--password <pass>]  Update a router profile
  profile use <name>    Set the default profile
  profile remove <name> Remove a profile and its stored password
  profile rename <old> <new>  Rename a profile
  uninstall             Remove binary, config, and stored credentials
  completion bash|zsh    Generate shell completion script
  version               Show version

Flags:
  --ip <addr>        Router IP address (overrides config)
  --password <pass>  Router admin password (overrides config)
  --profile <name>   Router profile name (overrides auto-detection)
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
	flag.StringVar(&profileFlag, "profile", "", "router profile name")
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
	case "profile":
		cmdProfile(args[1:])
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
					if err := ValidateChannel(*wifiChannel); err != nil {
						printError("%v", err)
						os.Exit(1)
					}
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
			if !jsonOutput {
				fmt.Print("are you sure? this will overwrite all config. type 'yes': ")
				var s string
				fmt.Scanln(&s)
				if s != "yes" {
					os.Exit(0)
				}
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
	var cfg *Config
	var profileName string
	var detected, cacheHit bool
	var fp NetworkFingerprint
	fpKey := ""

	if ip == "" || password == "" {
		var err error
		cfg, err = LoadConfig()
		if err != nil {
			printError("config error: %v", err)
			os.Exit(1)
		}

		if ip == "" {
			switch {
			case profileFlag != "":
				p, name, err := ActiveProfile(cfg, profileFlag)
				if err != nil {
					printError("unknown profile %q (have: %s)", profileFlag, profileNames(cfg))
					os.Exit(1)
				}
				if p == nil {
					printError("no such profile: %s", profileFlag)
					os.Exit(1)
				}
				profileName = name
				ip = p.IP
			default:
				fp, ferr := networkFingerprint()
				if ferr == nil {
					fpKey = fingerprintKey(fp)
				}
				name, _ := detectProfile(cfg, "", fp)
				if name != "" {
					if p, _, aerr := ActiveProfile(cfg, name); aerr == nil && p != nil {
						profileName = name
						ip = p.IP
						detected = true
						if fpKey != "" && cfg.NetworkCache[fpKey] == name {
							cacheHit = true
						}
					}
				}
				if profileName == "" {
					p, name, err := ActiveProfile(cfg, "")
					if err != nil {
						printError("%v", err)
						os.Exit(1)
					}
					if p != nil {
						profileName = name
						ip = p.IP
						detected = true
					}
				}
			}
			if ip != "" {
				if err := ValidateIPv4(ip); err != nil {
					printError("invalid IP in config: %v", err)
					os.Exit(1)
				}
			}
		} else {
			// --ip was given without --password; resolve the active profile so
			// a stored keyring password can be consulted below.
			if p, name, ok, aerr := lookupActiveProfile(cfg); aerr == nil && ok && p != nil {
				profileName = name
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
				if cfg == nil {
					cfg = emptyConfig()
				}
				cfg.Profiles["default"] = Profile{IP: ip}
				cfg.DefaultProfile = "default"
				if err := SaveConfig(cfg); err != nil {
					printError("%v", err)
					os.Exit(1)
				}
				profileName = "default"
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

	if password == "" && profileName != "" {
		var err error
		password, err = keyringGetPassword(profileName)
		if err != nil {
			printError("no password set for profile %q", profileName)
			if !jsonOutput {
				fmt.Fprintf(os.Stderr, "  set it:  tenda-n300 profile set %s --password <pass>\n  or run:   tenda-n300 --password <pass> <command>\n", profileName)
			}
			os.Exit(1)
		}
	}
	if password == "" {
		printError("router password required (use --password, `profile set`, or `config set password`)")
		os.Exit(1)
	}
	client, err := NewRouterClient(ip, password)
	if err != nil {
		if cacheHit && cfg != nil {
			delete(cfg.NetworkCache, fpKey)
			SaveConfig(cfg)
			if cfg.DefaultProfile != "" && cfg.DefaultProfile != profileName {
				if p, ok := cfg.Profiles[cfg.DefaultProfile]; ok {
					if pw, kerr := keyringGetPassword(cfg.DefaultProfile); kerr == nil {
						if c, cerr := NewRouterClient(p.IP, pw); cerr == nil {
							return c
						}
					}
				}
			}
		}
		printError("login failed: %v", err)
		os.Exit(1)
	}
	if detected && cfg != nil && recordNetworkProfile(cfg, fp, profileName) {
		SaveConfig(cfg)
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
	fmt.Printf("save %s as a profile? [y/N]: ", routers[0])
	var s string
	fmt.Scanln(&s)
	if s != "y" && s != "Y" && s != "yes" {
		return
	}
	name := "default"
	fmt.Printf("profile name [default]: ")
	var n string
	fmt.Scanln(&n)
	if n != "" {
		if err := validateProfileName(n); err != nil {
			printError("%v", err)
			os.Exit(1)
		}
		name = n
	}
	if err := ValidateIPv4(routers[0]); err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	cfg, err := LoadConfig()
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if _, exists := cfg.Profiles[name]; exists {
		printError("profile %q already exists", name)
		os.Exit(1)
	}
	cfg.Profiles[name] = Profile{IP: routers[0]}
	fmt.Printf("set as default? [y/N]: ")
	var d string
	fmt.Scanln(&d)
	if d == "y" || d == "Y" || d == "yes" {
		cfg.DefaultProfile = name
	}
	if err := SaveConfig(cfg); err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	fmt.Printf("added profile %q for %s\n", name, routers[0])
}

func cmdPing(ip string) {
	if ip == "" {
		cfg, err := LoadConfig()
		if err != nil {
			printError("config error: %v", err)
			os.Exit(1)
		}
		if p, _ := resolveActiveProfile(cfg); p != nil {
			ip = p.IP
		}
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

	// Remove keyring entries
	if err := keyringDeleteAllPasswords(); err != nil {
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
		p, name, _, _ := lookupActiveProfile(cfg)
		if jsonOutput {
			printJSON(showConfig(cfg))
			return
		}
		if p == nil {
			fmt.Println("active profile: <none>")
			fmt.Println("ip:             <not set>")
			fmt.Println("password:       <not set>")
			return
		}
		fmt.Printf("active profile: %s\n", name)
		fmt.Printf("ip:             %s\n", p.IP)
		if _, err := keyringGetPassword(name); err == nil {
			fmt.Println("password:       <set (stored in OS keyring)>")
		} else {
			fmt.Println("password:       <not set>")
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
			cfg = emptyConfig()
		}
		switch key {
		case "ip":
			if err := ValidateIPv4(val); err != nil {
				printError("%v", err)
				os.Exit(1)
			}
			_, name := resolveActiveProfile(cfg)
			if name == "" {
				name = "default"
				cfg.DefaultProfile = "default"
			}
			prof := cfg.Profiles[name]
			prof.IP = val
			cfg.Profiles[name] = prof
			if !jsonOutput {
				fmt.Fprintf(os.Stderr, "set ip %s for profile %q\n", val, name)
			}
		case "password":
			_, name := resolveActiveProfile(cfg)
			if err := keyringSetPassword(name, val); err != nil {
				printError("failed to save password to OS keyring: %v", err)
				os.Exit(1)
			}
			if !jsonOutput {
				if name != "" {
					fmt.Fprintf(os.Stderr, "password saved to OS keyring for profile %q\n", name)
				} else {
					fmt.Fprintln(os.Stderr, "password saved to OS keyring")
				}
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

func cmdProfile(args []string) {
	// --json/--profile may appear anywhere in the args (e.g.
	// `profile add home --json`); they only affect output mode and the active
	// profile, never which subcommand runs. Strip them before dispatching and
	// restore the globals afterwards.
	oldJSON := jsonOutput
	oldProfile := profileFlag
	args = extractGlobalFlags(args)
	defer func() {
		jsonOutput = oldJSON
		profileFlag = oldProfile
	}()

	for _, a := range args {
		if a == "--help" || a == "-h" {
			printProfileHelp()
			os.Exit(0)
		}
	}
	if len(args) == 0 {
		profileShow()
		return
	}
	switch args[0] {
	case "list":
		profileList()
	case "add":
		profileAdd(args[1:])
	case "set":
		profileSet(args[1:])
	case "use":
		profileUse(args[1:])
	case "remove":
		profileRemove(args[1:])
	case "rename":
		profileRename(args[1:])
	default:
		printError("unknown profile subcommand: %s", args[0])
		if !jsonOutput {
			printProfileHelp()
		}
		os.Exit(1)
	}
}

// extractGlobalFlags removes --json/--profile flags (in either position) from
// args, applies them to the package globals, and returns the remaining
// positional args.
func extractGlobalFlags(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			jsonOutput = true
		case strings.HasPrefix(a, "--json="):
			jsonOutput = a != "--json=false"
		case a == "--profile":
			if i+1 < len(args) {
				i++
				profileFlag = args[i]
			}
		case strings.HasPrefix(a, "--profile="):
			profileFlag = strings.TrimPrefix(a, "--profile=")
		default:
			out = append(out, a)
		}
	}
	return out
}

// resolveActiveProfile returns the profile selected by --profile or the
// configured default. A nil profile with empty name means no profile is
// configured yet. Exits on ambiguous/default-missing errors; use
// lookupActiveProfile for paths that must tolerate no active profile.
func resolveActiveProfile(cfg *Config) (*Profile, string) {
	p, name, ok, err := lookupActiveProfile(cfg)
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if !ok {
		return nil, ""
	}
	return p, name
}

// lookupActiveProfile resolves the profile selected by --profile or the
// configured default without exiting. ok is false when no profile is
// configured; err is non-nil when the selection is ambiguous (multiple
// profiles, no default) or invalid.
func lookupActiveProfile(cfg *Config) (p *Profile, name string, ok bool, err error) {
	if profileFlag != "" {
		p, name, err := ActiveProfile(cfg, profileFlag)
		if err != nil {
			return nil, "", false, err
		}
		return p, name, true, nil
	}
	p, name, err = ActiveProfile(cfg, "")
	if err != nil {
		return nil, "", false, err
	}
	if p == nil {
		return nil, "", false, nil
	}
	return p, name, true, nil
}

// showConfig is the JSON representation printed by `config` (show). It omits
// the internal network cache.
func showConfig(cfg *Config) Config {
	if cfg == nil {
		return Config{Profiles: map[string]Profile{}}
	}
	return Config{
		DefaultProfile: cfg.DefaultProfile,
		Profiles:       cfg.Profiles,
	}
}

func sortedProfileNames(cfg *Config) []string {
	names := make([]string, 0, len(cfg.Profiles))
	for n := range cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func profileShow() {
	cfg, err := LoadConfig()
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	_, active, _, _ := lookupActiveProfile(cfg)
	names := sortedProfileNames(cfg)
	if jsonOutput {
		printJSON(map[string]any{
			"default":  cfg.DefaultProfile,
			"active":   active,
			"profiles": names,
		})
		return
	}
	fmt.Printf("default: %s\n", cfg.DefaultProfile)
	if active == "" {
		fmt.Println("active: <none>")
	} else {
		fmt.Printf("active: %s\n", active)
	}
	if len(names) == 0 {
		fmt.Println("profiles: <none>")
	} else {
		fmt.Printf("profiles: %s\n", strings.Join(names, ", "))
	}
}

func profileList() {
	cfg, err := LoadConfig()
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	names := sortedProfileNames(cfg)
	if jsonOutput {
		printJSON(names)
		return
	}
	for _, n := range names {
		fmt.Println(n)
	}
}

// parseProfileFlags extracts --ip/--password values from args while leaving
// positional arguments in rest. Unlike flag.FlagSet it tolerates flags after
// positionals (e.g. `profile add home --ip 192.168.0.1`). `--` terminates flag
// parsing; a flag that needs a value errors instead of swallowing the next
// token (which may itself be a flag).
func parseProfileFlags(args []string) (ip, password string, rest []string, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			rest = append(rest, args[i+1:]...)
			return ip, password, rest, nil
		}
		switch {
		case a == "--ip" || a == "--password":
			flagName := strings.TrimPrefix(a, "--")
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				return ip, password, rest, fmt.Errorf("flag needs an argument: --%s", flagName)
			}
			i++
			if a == "--ip" {
				ip = args[i]
			} else {
				password = args[i]
			}
		case strings.HasPrefix(a, "--ip="):
			ip = strings.TrimPrefix(a, "--ip=")
		case strings.HasPrefix(a, "--password="):
			password = strings.TrimPrefix(a, "--password=")
		default:
			rest = append(rest, a)
		}
	}
	return ip, password, rest, nil
}

func profileAdd(args []string) {
	ip, pwd, rest, err := parseProfileFlags(args)
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if len(rest) != 1 {
		printError("usage: tenda-n300 profile add <name> [--ip <addr>] [--password <pass>]")
		os.Exit(1)
	}
	name := rest[0]
	if err := validateProfileName(name); err != nil {
		printError("%v", err)
		os.Exit(1)
	}

	cfg, err := LoadConfig()
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if _, exists := cfg.Profiles[name]; exists {
		printError("profile %q already exists (use `profile set`)", name)
		os.Exit(1)
	}

	if ip == "" {
		if jsonOutput {
			printError("no IP provided (use --ip)")
			os.Exit(1)
		}
		fmt.Printf("Router IP for profile %q: ", name)
		fmt.Scanln(&ip)
	}
	if ip == "" {
		printError("no IP provided (use --ip)")
		os.Exit(1)
	}
	if err := ValidateIPv4(ip); err != nil {
		printError("%v", err)
		os.Exit(1)
	}

	if pwd == "" && !jsonOutput {
		fmt.Printf("Router password for profile %q: ", name)
		fmt.Scanln(&pwd)
	}
	if pwd != "" {
		if err := keyringSetPassword(name, pwd); err != nil {
			printError("failed to save password to OS keyring: %v", err)
			os.Exit(1)
		}
	}

	cfg.Profiles[name] = Profile{IP: ip}
	if len(cfg.Profiles) == 1 {
		if jsonOutput {
			cfg.DefaultProfile = name
		} else {
			fmt.Printf("set %q as the default profile? [y/N]: ", name)
			var s string
			fmt.Scanln(&s)
			if s == "y" || s == "Y" || s == "yes" {
				cfg.DefaultProfile = name
			}
		}
	}
	if err := SaveConfig(cfg); err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if jsonOutput {
		printJSON(map[string]any{"status": "ok", "profile": name, "ip": ip, "default": cfg.DefaultProfile == name})
	} else {
		def := ""
		if cfg.DefaultProfile == name {
			def = " (default)"
		}
		fmt.Printf("added profile %q%s\n", name, def)
	}
}

func profileSet(args []string) {
	ip, pwd, rest, err := parseProfileFlags(args)
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if len(rest) != 1 {
		printError("usage: tenda-n300 profile set <name> [--ip <addr>] [--password <pass>]")
		os.Exit(1)
	}
	if ip == "" && pwd == "" {
		printError("nothing to change: pass at least one of --ip or --password")
		os.Exit(1)
	}
	name := rest[0]
	cfg, err := LoadConfig()
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	p, ok := cfg.Profiles[name]
	if !ok {
		printError("profile %q does not exist (have: %s)", name, profileNames(cfg))
		os.Exit(1)
	}
	if ip != "" {
		if err := ValidateIPv4(ip); err != nil {
			printError("%v", err)
			os.Exit(1)
		}
		p.IP = ip
		cfg.Profiles[name] = p
		if err := SaveConfig(cfg); err != nil {
			printError("%v", err)
			os.Exit(1)
		}
	}
	if pwd != "" {
		if err := keyringSetPassword(name, pwd); err != nil {
			printError("failed to save password to OS keyring: %v", err)
			os.Exit(1)
		}
	}
	if jsonOutput {
		printJSON(map[string]any{"status": "ok", "profile": name, "ip": p.IP})
	} else {
		fmt.Printf("updated profile %q\n", name)
	}
}

func profileUse(args []string) {
	if len(args) < 1 {
		printError("usage: tenda-n300 profile use <name>")
		os.Exit(1)
	}
	name := args[0]
	cfg, err := LoadConfig()
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if _, ok := cfg.Profiles[name]; !ok {
		printError("profile %q does not exist (have: %s)", name, profileNames(cfg))
		os.Exit(1)
	}
	cfg.DefaultProfile = name
	if err := SaveConfig(cfg); err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if jsonOutput {
		printJSON(map[string]string{"status": "ok", "default_profile": name})
	} else {
		fmt.Printf("default profile set to %q\n", name)
	}
}

func profileRemove(args []string) {
	if len(args) < 1 {
		printError("usage: tenda-n300 profile remove <name>")
		os.Exit(1)
	}
	name := args[0]
	cfg, err := LoadConfig()
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if _, ok := cfg.Profiles[name]; !ok {
		printError("profile %q does not exist (have: %s)", name, profileNames(cfg))
		os.Exit(1)
	}
	delete(cfg.Profiles, name)
	for k, v := range cfg.NetworkCache {
		if v == name {
			delete(cfg.NetworkCache, k)
		}
	}
	if cfg.DefaultProfile == name {
		switch len(cfg.Profiles) {
		case 0:
			cfg.DefaultProfile = ""
		case 1:
			for n := range cfg.Profiles {
				cfg.DefaultProfile = n
			}
		default:
			cfg.DefaultProfile = ""
		}
	}
	keyringDeletePassword(name)
	if err := SaveConfig(cfg); err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if jsonOutput {
		printJSON(map[string]any{"status": "ok", "removed": name, "default_profile": cfg.DefaultProfile})
	} else {
		fmt.Printf("removed profile %q\n", name)
	}
}

func profileRename(args []string) {
	if len(args) < 2 {
		printError("usage: tenda-n300 profile rename <old> <new>")
		os.Exit(1)
	}
	oldName, newName := args[0], args[1]
	if err := validateProfileName(newName); err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	cfg, err := LoadConfig()
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	p, ok := cfg.Profiles[oldName]
	if !ok {
		printError("profile %q does not exist (have: %s)", oldName, profileNames(cfg))
		os.Exit(1)
	}
	if _, exists := cfg.Profiles[newName]; exists {
		printError("profile %q already exists", newName)
		os.Exit(1)
	}
	if pwd, err := keyringGetPassword(oldName); err == nil {
		if err := keyringSetPassword(newName, pwd); err != nil {
			printError("failed to save password to OS keyring: %v", err)
			os.Exit(1)
		}
		keyringDeletePassword(oldName)
	}
	delete(cfg.Profiles, oldName)
	cfg.Profiles[newName] = p
	for k, v := range cfg.NetworkCache {
		if v == oldName {
			cfg.NetworkCache[k] = newName
		}
	}
	if cfg.DefaultProfile == oldName {
		cfg.DefaultProfile = newName
	}
	if err := SaveConfig(cfg); err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if jsonOutput {
		printJSON(map[string]string{"status": "ok", "renamed": oldName, "to": newName})
	} else {
		fmt.Printf("renamed profile %q to %q\n", oldName, newName)
	}
}

func printProfileHelp() {
	fmt.Fprintf(os.Stderr, `Usage: tenda-n300 profile [command]

Manage router profiles.

Subcommands:
  list                 List profile names
  add <name> [--ip <addr>] [--password <pass>]
                       Add a new profile (prompts for missing values)
  set <name> [--ip <addr>] [--password <pass>]
                       Update an existing profile (at least one flag required)
  use <name>           Set the default profile
  remove <name>        Remove a profile and its stored password
  rename <old> <new>   Rename a profile (moves its stored password)

With no subcommand, shows the default and active profile.
`)
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
		"profile":      "Usage: tenda-n300 profile [list|add <name> [--ip <addr>] [--password <pass>]|set <name> [--ip <addr>] [--password <pass>]|use <name>|remove <name>|rename <old> <new>]\n\nManage router profiles.",
	}
	if h, ok := help[cmd]; ok {
		fmt.Fprintln(os.Stderr, h)
	} else {
		fmt.Fprintf(os.Stderr, "no help available for %s\n", cmd)
	}
}
