# tenda-n300

A command-line tool for controlling a **Tenda N300** wireless router from your terminal. No browser needed.

> **Disclaimer:** This tool interacts with the router's internal web API. It is not officially supported by Tenda. Use at your own risk.

## Features

- **List devices** — view all connected and blocked devices on the network
- **Block / unblock** — restrict internet access for any device by MAC address
- **Reboot** — restart the router remotely
- **Factory reset** — wipe all router settings (with interactive confirmation)
- **Backup / restore** — save and reload router configuration
- **Syslog** — export the router's system log
- **Discover** — scan the local network for Tenda routers
- **JSON output** — machine-readable output for scripting (`--json`)
- **Shell completion** — bash and zsh tab completion

## Installation

### Prerequisites

- [Go](https://go.dev/dl/) 1.25 or later
- A Tenda N300 router on the same network

### Option A: Quick install

```sh
git clone https://github.com/kakashy/tenda-n300.git
cd tenda-n300
./install.sh
```

This builds the binary and installs it to `/usr/local/bin` (or `~/.local/bin`).

### Option B: Manual build

```sh
git clone https://github.com/kakashy/tenda-n300.git
cd tenda-n300
go build -ldflags="-s -w" -o tenda-n300 .
sudo mv tenda-n300 /usr/local/bin/
```

> Or just use `go install` if `$GOPATH/bin` is on your `$PATH`:
>
> ```sh
> go install github.com/kakashy/tenda-n300@latest
> ```

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

| Flag                | Description                              |
| ------------------- | ---------------------------------------- |
| `--ip <addr>`       | Router IP address (overrides config)     |
| `--password <pass>` | Router admin password (overrides config) |
| `--json`            | Output in JSON format                    |

### Examples

```sh
# List devices with JSON output
tenda-n300 --json devices

# Reboot using inline credentials
tenda-n300 --ip 192.168.0.1 --password admin reboot

# Save syslog to a file
tenda-n300 syslog /tmp/router.log
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

### From a pre-built binary

```sh
tenda-n300 uninstall
```

Both methods remove the binary, shell completions, config file, and stored credentials.

## License

[MIT](LICENSE)
