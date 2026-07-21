# tenda-n300

A command-line tool for controlling a **Tenda N300** wireless router from your terminal. No browser needed.

> **Disclaimer:** This tool interacts with the router's internal web API. It is not officially supported by Tenda. Use at your own risk.

## Features

- **List devices** — view all connected and blocked devices on the network
- **Block / unblock** — restrict internet access for any device by MAC address
- **WiFi settings** — view or change SSID, password, channel, and encryption mode
- **Reboot** — restart the router remotely
- **Factory reset** — wipe all router settings (with interactive confirmation)
- **Backup / restore** — save and reload router configuration
- **Syslog** — export the router's system log
- **Discover** — scan the local network for Tenda routers
- **JSON output** — machine-readable output for scripting (`--json`)
- **Shell completion** — bash and zsh tab completion

## Installation

### Quick install (one-liner, no Go needed)

```sh
curl -fsSL https://raw.githubusercontent.com/kakashy/tenda-n300/main/install.sh | bash
```

Downloads the latest pre-built binary for your OS and architecture and installs it to `/usr/local/bin` (or `~/.local/bin`).

### Build from source

Requires [Go](https://go.dev/dl/) 1.25 or later.

```sh
git clone https://github.com/kakashy/tenda-n300.git
cd tenda-n300
./build.sh
```

### Manual build

```sh
git clone https://github.com/kakashy/tenda-n300.git
cd tenda-n300
go build -ldflags="-s -w" -o tenda-n300 .
sudo mv tenda-n300 /usr/local/bin/
```

### Go install

```sh
go install github.com/kakashy/tenda-n300@latest
```

## Quick start

1. **Discover** your router on the network:

   ```sh
   tenda-n300 discover
   ```

2. **Save** the router IP and admin password to config:

   ```sh
   tenda-n300 config set ip 192.168.0.1
   tenda-n300 config set password admin
   ```

3. **List** connected devices:

   ```sh
   tenda-n300 devices
   ```

4. **Block** a device:

   ```sh
   tenda-n300 block AA:BB:CC:DD:EE:FF
   ```

## Usage

```
tenda-n300 [--ip <addr>] [--password <pass>] [--json] <command> [args]
```

### Commands

| Command                | Description                                                    |
| ---------------------- | -------------------------------------------------------------- |
| `devices`              | List connected and blocked devices                             |
| `block <mac>`          | Block a device by MAC address                                  |
| `unblock <mac>`        | Unblock a device by MAC address                                |
| `wifi`                 | Show current WiFi settings (SSID, password, channel, encryption) |
| `wifi --ssid <n> --wifi-password <p> --channel <c> --encrypt <e>` | Change WiFi settings (any combination of flags) |
| `status`               | Show router summary (total / online / blocked)                 |
| `reboot`               | Reboot the router                                              |
| `reset`                | Factory reset (wipes all config — requires `yes` confirmation) |
| `backup [file]`        | Download config backup (defaults to `RouterCfm.cfg`)           |
| `restore <file>`       | Restore config from a backup file                              |
| `syslog [file]`        | Export the system log (prints to stdout if no file given)      |
| `discover`             | Scan LAN for Tenda routers                                     |
| `config`               | Show or set persistent config (`ip`, `password`)               |
| `completion bash\|zsh` | Generate shell completion script                               |
| `uninstall`            | Remove binary, config, and stored credentials                  |

### Flags

| Flag                      | Description                              |
| ------------------------- | ---------------------------------------- |
| `--ip <addr>`             | Router IP address (overrides config)     |
| `--password <pass>`       | Router admin password (overrides config) |
| `--json`                  | Output in JSON format                    |
| `--ssid <name>`           | New WiFi SSID (for `wifi` command)       |
| `--wifi-password <pass>`  | New WiFi password (for `wifi` command)   |
| `--channel <n>`           | New WiFi channel 1-11 (for `wifi` command) |
| `--encrypt <mode>`        | New WiFi encryption mode (for `wifi` command) |

### Examples

```sh
# List devices with JSON output
tenda-n300 --json devices

# Reboot using inline credentials
tenda-n300 --ip 192.168.0.1 --password admin reboot

# Save syslog to a file
tenda-n300 syslog /tmp/router.log

# Show current WiFi settings
tenda-n300 wifi

# Change WiFi SSID and password
tenda-n300 wifi --ssid "MyNetwork" --wifi-password "newpass123"

# Change WiFi channel to 11 with WPA2 encryption
tenda-n300 wifi --channel 11 --encrypt "WPA2PSK/AES"
```

## How it works

The tool communicates with the Tenda N300's proprietary goform API over HTTP:

- **Authentication:** MD5 challenge-response (`MD5(base64(password) + token)`)
- **Device management:** QoS goform endpoints (`getQos`, `setQos`)
- **System actions:** `sysReboot`, `sysRestore`
- **Config / logs:** CGI endpoints (`DownloadCfg`, `UploadCfg`, `DownloadSyslog`)

Auto-discovery (`discover` command) sweeps the local /24 subnet looking for HTTP servers whose login page contains `"tenda"`.

## Security

- The router admin password is stored in your **OS keyring** (macOS Keychain, Linux Secret Service, Windows Credential Manager) — never in plaintext on disk.
- All communication with the router is over **plain HTTP** — no TLS.
- See [`SECURITY.md`](SECURITY.md) for details.

## Uninstall

### From source

```sh
./uninstall.sh
```

> If you installed via `build.sh`, the binary is at `~/.local/bin/tenda-n300`.

### From a pre-built binary

```sh
tenda-n300 uninstall
```

Both methods remove the binary, shell completions, config file, and stored credentials.

## License

[MIT](LICENSE)
