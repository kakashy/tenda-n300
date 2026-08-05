# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Plain static HTML/CSS/JS, no build step, deployed to GitHub Pages (user-confirmed choice).

## Users

Primary audience: homelab and sysadmin enthusiasts — technical people managing home or small-office networks who prefer the terminal over router web UIs. They discover the tool, want to understand what it does in seconds, and decide to install it. Secondary audience: developers who may build from source or contribute.

## Product Purpose

`tenda-n300` is a command-line tool for controlling a Tenda N300 wireless router from the terminal — listing devices, blocking/unblocking by MAC, changing WiFi settings, rebooting, resetting, backup/restore, syslog, and multi-router profiles — with no browser needed. The site's job is to make the tool instantly intelligible, credible, and installable for that audience.

## Positioning

The meaningful mechanism a neighboring product could not copy: direct control of a cheap, ubiquitous consumer router through the router's own HTTP/goform API, wrapped in a fast, scriptable CLI with profiles, auto-detection, and JSON output. No browser, no vendor app, no cloud.

## Operating Context

Visitors are on their own networks or configuring home infra. They copy-paste shell commands (install one-liner, go install, brew-style flows), expect terminal-accurate command examples, and value scriptability (JSON output), security hygiene (passwords stored in the OS keyring), and the disclaimer that this is an unofficial tool used at one's own risk.

## Capabilities and Constraints

Confirmed capabilities (from README): list devices, block/unblock by MAC, WiFi settings view/change (SSID, password, channel, encryption), reboot, factory reset, backup/restore, syslog export, profiles with auto-detection, LAN discovery, JSON output, shell completion, uninstall. Technical facts: communicates over the router's plain-HTTP goform API; MD5 challenge-response auth; passwords in OS keyring; requires Go 1.25+ to build; MIT licensed. Constraint: not officially supported by Tenda — must not imply official endorsement. No screenshots, logos, or product imagery exist yet; any demonstration imagery is authorable as synthetic, factual claims are not.

## Brand Commitments

No existing brand assets. Name is `tenda-n300`. Must avoid implying official Tenda branding or using their logo. User granted free rein on visual identity.

## Evidence on Hand

README.md (features, install flows, full command reference, examples, security notes), install.sh (curl-pipe-bash one-liner), LICENSE (MIT). No testimonials, customers, or press — must not fabricate.

## Product Principles

- Honest about what it is: an unofficial, terminal-first tool, clearly framed.
- Technical credibility first: exact commands, real flags, real output — the audience trusts demonstrated capability, not adjectives.
- Fast path to value: install, discover, use — minimize steps between reading and doing.
- Respect the audience's environment: security hygiene (keyring, no plaintext), scriptability (JSON), and platform coverage are features, not footnotes.

## Accessibility & Inclusion

No product-specific requirement established. Site should follow standard accessibility practice (semantic HTML, contrast, keyboard usability).
