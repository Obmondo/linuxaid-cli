# linuxaid-cli

> **Part of [Linuxaid](https://github.com/obmondo/linuxaid) by [Obmondo](https://obmondo.com)**

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/Obmondo/linuxaid-cli)

A Linux system management toolkit that automates Puppet-based configuration management and system updates across Obmondo-managed infrastructure. The project provides two command-line applications for initial provisioning and ongoing system maintenance.

## Installation

### Quick Install (Recommended)

The easiest way to install `linuxaid-install` is using the installation script:

```bash
curl -sSL https://raw.githubusercontent.com/Obmondo/linuxaid-cli/main/install.sh | bash
```

This script will:

- Detect your system architecture (amd64, arm64, armv7)
- Download the latest release from GitHub
- Verify the checksum for security
- Install the `linuxaid-install` binary to `/usr/local/bin`
- Automatically run the installation tool

### Build from Source

If you prefer to build from source:

```bash
make build
```

This produces two binaries: `linuxaid-install` and `linuxaid-cli`.

## Applications

### linuxaid-install

Initial system provisioning tool that installs and configures the Puppet agent (openvox-agent). Obmondo customers authenticate with the `TOKEN` environment variable; opensource users can run it without a token.

**Usage:**

```bash
# Obmondo customers
TOKEN='your-token' linuxaid-install --certname web01.example --puppet-server your.openvoxserver.com --openvox-environment master

# Opensource users (no token required)
linuxaid-install --certname web01.example --puppet-server your.openvoxserver.com --openvox-environment master
```

### linuxaid-cli

System management CLI for Puppet-managed nodes with commands for automated maintenance operations.

**Commands:**

- `system-update` - Executes distribution-specific package updates with service window coordination
- `run-openvox` - Runs Puppet agent with connectivity checks and status reporting

**Usage:**

```bash
linuxaid-cli system-update --certname web01.example --no-reboot
linuxaid-cli run-openvox --certname web01.example
```

## Features

- **Automated Puppet Agent Management**: Complete lifecycle management including installation, configuration, and execution
- **Remote Logging**: gRPC-based streaming of command execution output to Obmondo API
- **Service Window Coordination**: Integrates with Obmondo backend for maintenance window management
- **Multi-Distribution Support**: Ubuntu, Debian, RHEL, CentOS, SLES, OracleLinux, and TurrisOS

## Configuration

Configuration uses a three-tier precedence system via Viper:

1. Command-line flags (highest priority)
2. Environment variables
3. Default values

## Global Flags

These flags are available for both `linuxaid-install` and `linuxaid-cli`:

- `--certname` / `CERTNAME` - Certificate name (required)
- `--debug` / `DEBUG` - Enable debug logging

## `linuxaid-install` Specific Flags

- `--puppet-server` / `PUPPET_SERVER` - Puppet server hostname
- `--openvox-environment` / `OPENVOX_ENVIRONMENT` - Openvox environment (Linuxaid release version)

## `linuxaid-cli` Command-Specific Flags

### system-update

- `--no-reboot` / `NO_REBOOT` - Disable automatic reboot after updates
- `--skip-openvox` / `SKIP_OPENVOX` - Skip Puppet agent run
- `--security-exporter-url` / `SECURITY_EXPORTER_URL` - URL of the security exporter (default: `http://127.254.254.254:63396`)

### run-openvox

No command-specific flags available. Uses global flags only.

## Dependencies

- Cobra for CLI framework
- Viper for configuration management
- gRPC for remote logging
- Bitfield/script for shell command execution
