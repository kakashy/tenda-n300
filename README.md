# tenda-n300

A command-line tool for controlling a **Tenda N300** wireless router from your terminal. No browser needed.

> **Website:** <https://kakashy.github.io/tenda-n300/>

> **Disclaimer:** This tool interacts with the router's internal web API. It is not officially supported by Tenda. Use at your own risk.

## Features

- **List devices** — view all connected and blocked devices on the network
- **Block / unblock** — restrict internet access for any device by MAC address
- **WiFi settings** — view or change SSID, password, channel, and encryption mode
- **Reboot** — restart the router remotely
- **Factory reset** — wipe all router settings (with interactive confirmation)
- **Backup / restore** — save and reload router configuration
- **Syslog** — export the router's system log
- **Profiles** — manage multiple routers (home, work) and switch between them
- **Auto-detection** — remembers which router you're connected to and picks the right profile
- **Discover** — scan the local network for Tenda routers and save them as profiles
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

2. **Save** the router IP and admin password as a profile and make it the default:

   ```sh
   tenda-n300 profile add home --ip 192.168.0.1 --password admin
   tenda-n300 profile use home
   ```

3. **List** connected devices:

   ```sh
   tenda-n300 devices
   ```

4. **Block** a device:

   ```sh
   tenda-n300 block AA:BB:CC:DD:EE:FF
   ```

> Prefer `profile add home --ip 192.168.0.1 --password admin` over the older
> `config set ip` / `config set password` commands — `config set` now operates
> on the *active* profile (the `--profile` flag or the default).

## Usage

```
tenda-n300 [--ip <addr>] [--password <pass>] [--profile <name>] [--json] <command> [args]
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
| `discover`             | Scan LAN for Tenda routers (can save a found router as a profile) |
| `config`               | Show active profile / set `ip` or `password` (applies to the active profile) |
| `profile`              | Show active and default profiles                               |
| `profile list`         | List profile names                                             |
| `profile add <name> [--ip <addr>] [--password <pass>]` | Add a router profile                              |
| `profile set <name> [--ip <addr>] [--password <pass>]` | Update a router profile (at least one flag)                  |
| `profile use <name>`   | Set the default profile                                        |
| `profile remove <name>`| Remove a profile and its stored password                       |
| `profile rename <old> <new>` | Rename a profile (moves its stored password)             |
| `completion bash\|zsh` | Generate shell completion script                               |
| `uninstall`            | Remove binary, config, and stored credentials                  |

### Flags

| Flag                      | Description                              |
| ------------------------- | ---------------------------------------- |
| `--ip <addr>`             | Router IP address (overrides config and profiles) |
| `--password <pass>`       | Router admin password (overrides config and profiles) |
| `--profile <name>`        | Router profile name (overrides auto-detection) |
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

### Multiple routers (work/home)

Keep one profile per router and switch between them:

```sh
# add both routers
tenda-n300 profile add home --ip 192.168.0.1 --password admin
tenda-n300 profile add work --ip 10.0.0.1 --password secret

# switch the default
tenda-n300 profile use work

# use a specific profile for a single command
tenda-n300 --profile home devices

# list and inspect profiles
tenda-n300 profile list
tenda-n300 profile
```

**Auto-detection:** the tool fingerprints the current network (default gateway
IP + MAC). On networks it has seen before it picks the matching profile
automatically; a successful login on a new network remembers it for next time.
When no profile matches, the default profile is used, or you can pin one with
`--profile`. `config set ip` / `config set password` always apply to the active
profile (`--profile` or the default).

## How it works

The tool communicates with the Tenda N300's proprietary goform API over HTTP:

- **Authentication:** MD5 challenge-response (`MD5(base64(password) + token)`)
- **Device management:** QoS goform endpoints (`getQos`, `setQos`)
- **System actions:** `sysReboot`, `sysRestore`
- **Config / logs:** CGI endpoints (`DownloadCfg`, `UploadCfg`, `DownloadSyslog`)

Auto-discovery (`discover` command) sweeps the local /24 subnet looking for HTTP servers whose login page contains `"tenda"`.

## Security

- The router admin password is stored in your **OS keyring** (macOS Keychain, Linux Secret Service, Windows Credential Manager) — never in plaintext on disk. Passwords are keyed per profile: the default profile uses the key `password`, named profiles use `password:<name>`.
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
