package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// cmdPortForward manages port forwarding (virtual server) rules:
//
//	tenda-n300 portforward                          list rules
//	tenda-n300 portforward add <ip> <in-port> <ext-port> [--protocol tcp|udp|both]
//	tenda-n300 portforward remove <index>
//
// The N300 firmware stores exactly four fields per rule (internal IP,
// internal port, external port, protocol) and has no rule names, so removal
// is by the 1-based index shown by `list`.
func cmdPortForward(args []string, ip, password string) {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			portForwardHelp()
			os.Exit(0)
		}
	}
	if len(args) == 0 || args[0] == "list" {
		portForwardList(ip, password)
		return
	}
	switch args[0] {
	case "add":
		portForwardAdd(args[1:], ip, password)
	case "remove":
		portForwardRemove(args[1:], ip, password)
	default:
		printError("unknown portforward subcommand: %s", args[0])
		if !jsonOutput {
			portForwardHelp()
		}
		os.Exit(1)
	}
}

func portForwardList(ip, password string) {
	client := connectRouter(ip, password)
	startSpinner("fetching port forwarding rules")
	nat, err := client.GetNAT()
	stopSpinner()
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	printPortForwardRules(nat.PortRules)
}

// portForwardAdd fetches the current rules, validates the new rule against
// the same constraints as the router web UI (valid IP/ports, on-LAN subnet,
// no duplicate external port), appends it, and pushes the whole list back.
func portForwardAdd(args []string, ip, password string) {
	protocol, rest, err := parsePortForwardFlags(args)
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if len(rest) != 3 {
		usageError("usage: tenda-n300 portforward add <internal-ip> <internal-port> <external-port> [--protocol tcp|udp|both]")
	}
	internalIP, internalPort, externalPort := rest[0], rest[1], rest[2]

	if err := ValidateIPv4(internalIP); err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if err := ValidatePort(internalPort); err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if err := ValidatePort(externalPort); err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if err := ValidateProtocol(protocol); err != nil {
		printError("%v", err)
		os.Exit(1)
	}

	client := connectRouter(ip, password)
	startSpinner("fetching current port forwarding rules")
	nat, err := client.GetNAT()
	stopSpinner()
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}

	if nat.Lan != nil && nat.Lan.IP != "" && nat.Lan.Mask != "" {
		if !sameSubnet(internalIP, nat.Lan.IP, nat.Lan.Mask) {
			printError("internal IP %s is not on the router LAN subnet (%s/%s)", internalIP, nat.Lan.IP, nat.Lan.Mask)
			os.Exit(1)
		}
	}
	for _, r := range nat.PortRules {
		if r.ExternalPort == externalPort {
			printError("external port %s already has a rule (remove it first or pick another port)", externalPort)
			os.Exit(1)
		}
	}

	rules := append(nat.PortRules, PortForwardRule{
		InternalIP:   internalIP,
		InternalPort: internalPort,
		ExternalPort: externalPort,
		Protocol:     protocol,
	})
	startSpinner("updating port forwarding rules")
	err = client.SetPortForwardRules(rules)
	stopSpinner()
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}

	if jsonOutput {
		printJSON(map[string]any{
			"status": "ok",
			"rule": PortForwardRule{
				InternalIP:   internalIP,
				InternalPort: internalPort,
				ExternalPort: externalPort,
				Protocol:     protocol,
			},
		})
	} else {
		fmt.Printf("added port forwarding rule: %s:%s -> %s (%s)\n", externalPort, internalPort, internalIP, protocol)
	}
}

// portForwardRemove deletes the rule at the 1-based index shown by `list`.
func portForwardRemove(args []string, ip, password string) {
	if len(args) != 1 {
		usageError("usage: tenda-n300 portforward remove <index>")
	}
	idx, err := strconv.Atoi(args[0])
	if err != nil || idx < 1 {
		printError("invalid index %q (must be a positive number, see `portforward list`)", args[0])
		os.Exit(1)
	}

	client := connectRouter(ip, password)
	startSpinner("fetching current port forwarding rules")
	nat, err := client.GetNAT()
	stopSpinner()
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
	if idx > len(nat.PortRules) {
		printError("no rule at index %d (router has %d rule(s))", idx, len(nat.PortRules))
		os.Exit(1)
	}

	removed := nat.PortRules[idx-1]
	rules := append(nat.PortRules[:idx-1], nat.PortRules[idx:]...)
	startSpinner("updating port forwarding rules")
	err = client.SetPortForwardRules(rules)
	stopSpinner()
	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}

	if jsonOutput {
		printJSON(map[string]any{
			"status":  "ok",
			"removed": ruleJSON(idx, removed),
		})
	} else {
		fmt.Printf("removed rule %d: %s:%s -> %s (%s)\n", idx, removed.ExternalPort, removed.InternalPort, removed.InternalIP, removed.Protocol)
	}
}

func ruleJSON(index int, r PortForwardRule) map[string]string {
	return map[string]string{
		"index":         strconv.Itoa(index),
		"internal_ip":   r.InternalIP,
		"internal_port": r.InternalPort,
		"external_port": r.ExternalPort,
		"protocol":      r.Protocol,
	}
}

// parsePortForwardFlags extracts --protocol from args (in any position,
// mirroring parseProfileFlags) and returns the remaining positionals. The
// default protocol is "both", matching the router web UI.
func parsePortForwardFlags(args []string) (protocol string, rest []string, err error) {
	protocol = "both"
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			rest = append(rest, args[i+1:]...)
			return protocol, rest, nil
		case a == "--protocol":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				return "", nil, fmt.Errorf("flag needs an argument: --protocol")
			}
			i++
			protocol = args[i]
		case strings.HasPrefix(a, "--protocol="):
			protocol = strings.TrimPrefix(a, "--protocol=")
		default:
			rest = append(rest, a)
		}
	}
	return protocol, rest, nil
}

// sameSubnet reports whether ip is on the LAN subnet defined by lanIP/lanMask.
// Both are parsed as IPv4; anything unparsable reports false so callers can
// decide how strict to be.
func sameSubnet(ip, lanIP, lanMask string) bool {
	a := net.ParseIP(ip).To4()
	b := net.ParseIP(lanIP).To4()
	m := net.ParseIP(lanMask).To4()
	if a == nil || b == nil || m == nil {
		return false
	}
	for i := 0; i < 4; i++ {
		if a[i]&m[i] != b[i]&m[i] {
			return false
		}
	}
	return true
}

func portForwardHelp() {
	fmt.Fprintf(os.Stderr, `Usage: tenda-n300 portforward [list]
       tenda-n300 portforward add <internal-ip> <internal-port> <external-port> [--protocol tcp|udp|both]
       tenda-n300 portforward remove <index>

Manage port forwarding (virtual server) rules. Lets you open ports for game
servers, webcams, or home automation without logging into the router web UI.

Subcommands:
  list                           Show all port forwarding rules
  add <ip> <in-port> <ext-port>  Add a rule (protocol defaults to both)
  remove <index>                 Remove a rule by its 1-based index from list

With no subcommand, lists the rules. The N300 firmware has no rule names,
so rules are addressed by index.
`)
}
