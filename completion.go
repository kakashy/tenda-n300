package main

import (
	"fmt"
	"os"
)

func cmdCompletion(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: tenda-n300 completion bash|zsh\n")
		os.Exit(1)
	}
	switch args[0] {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	default:
		fmt.Fprintf(os.Stderr, "unknown shell: %s (use bash or zsh)\n", args[0])
		os.Exit(1)
	}
}

const bashCompletion = `_tenda_n300() {
    local cur prev words cword
    _init_completion || return

    commands="devices block unblock firmwareinfo wifi status reboot reset backup restore syslog ping discover config uninstall completion version"
    config_subcommands="set"

    case $prev in
        tenda-n300)
            COMPREPLY=($(compgen -W "$commands --ip --password --json --version" -- "$cur"))
            ;;
        wifi)
            COMPREPLY=($(compgen -W "--ssid --wifi-password --channel --encrypt --help" -- "$cur"))
            ;;
        config)
            COMPREPLY=($(compgen -W "set --help" -- "$cur"))
            ;;
        completion)
            COMPREPLY=($(compgen -W "bash zsh" -- "$cur"))
            ;;
        set)
            COMPREPLY=($(compgen -W "ip password" -- "$cur"))
            ;;
        block|unblock)
            # suggest MAC addresses from devices output (accepts multiple)
            COMPREPLY=()
            ;;
        backup|syslog)
            COMPREPLY=($(compgen -f -- "$cur"))
            ;;
        restore)
            COMPREPLY=($(compgen -f -- "$cur"))
            ;;
        *)
            if [[ $cur == -* ]]; then
                COMPREPLY=($(compgen -W "--ip --password --json --version" -- "$cur"))
            fi
            ;;
    esac
} &&
complete -F _tenda_n300 tenda-n300
`

const zshCompletion = `#compdef tenda-n300

_tenda_n300() {
    local -a commands
    commands=(
        'devices:list connected devices'
		'block:block one or more devices by MAC address'
		'unblock:unblock one or more devices by MAC address'
        'firmwareinfo:show router firmware information'
        'wifi:show or change WiFi settings (SSID, password, channel, encryption)'
        'status:show router summary'
        'reboot:reboot the router'
        'reset:factory reset router'
        'backup:download config backup'
        'restore:restore config from backup file'
        'syslog:export system log'
        'ping:check if router is reachable and responsive'
		'discover:scan network for Tenda routers'
		'config:show or set configuration'
		'uninstall:remove binary, config, and stored credentials'
		'completion:generate shell completion script'
		'version:show version'
    )

    _arguments -C \
        '--ip[router IP address]' \
        '--password[router admin password]' \
        '--json[output as JSON]' \
        '--version[show version]' \
        '1:command:->cmds' \
        '*::args:->args'

    case $state in
        cmds)
            _describe 'command' commands
            ;;
        args)
            case $words[1] in
                wifi)
                    _values 'option' \
                        '--ssid[new WiFi SSID]' \
                        '--wifi-password[new WiFi password]' \
                        '--channel[new WiFi channel (1-11)]' \
                        '--encrypt[new WiFi encryption mode]'
                    ;;
                config)
                    if [[ $CURRENT -eq 3 ]]; then
                        _values 'subcommand' 'set'
                    elif [[ $CURRENT -eq 4 ]]; then
                        _values 'key' 'ip' 'password'
                    fi
                    ;;
                block|unblock)
                    _message 'MAC address (e.g. aa:bb:cc:dd:ee:ff) — accepts multiple'
                    ;;
                completion)
                    _values 'shell' 'bash' 'zsh'
                    ;;
                backup|syslog|restore)
                    _files
                    ;;
            esac
            ;;
    esac
}

compdef _tenda_n300 tenda-n300
`
