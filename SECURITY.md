# Security Policy

## Known security considerations

This tool interacts with a Tenda N300 router over a local network. The following are inherent limitations of the router's design:

- **No TLS** — All communication with the router is over plain HTTP. Credentials and session tokens are transmitted in cleartext. Use only on trusted networks.
- **Password storage** — The router admin password is stored in your OS keyring (macOS Keychain, Linux Secret Service, Windows Credential Manager), not in a file on disk.
- **Weak authentication** — The router uses an MD5-based challenge-response scheme. This is not cryptographically strong by modern standards.
- **Local network only** — This tool is designed for use on your local LAN. Exposing the router's web interface to the internet is strongly discouraged regardless of this tool.

## Reporting a vulnerability

If you discover a security issue, please open an issue on GitHub rather than emailing maintainers directly. Do not disclose vulnerabilities publicly until they have been addressed.
