# linuxaid-cli

> **Part of [Linuxaid](https://github.com/obmondo/linuxaid) by [Obmondo](https://obmondo.com)**

A Linux system management toolkit that automates Puppet-based configuration management and system updates across Obmondo-managed infrastructure. The project provides two command-line applications for initial provisioning and ongoing system maintenance.

## Applications

### linuxaid-install

Initial system provisioning tool that installs and configures the Puppet agent (openvox-agent). Requires an `TOKEN` environment variable for authentication.

**Usage:**

```bash
TOKEN='your-token'
linuxaid-install --certname web01.example --puppet-server your.openvoxserver.com
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
- **Multi-Distribution Support**: Ubuntu, Debian, RHEL, CentOS, SLES, and TurrisOS

## Building

```bash
make build
```

This produces two binaries: `linuxaid-install` and `linuxaid-cli`.

## Configuration

Configuration uses a three-tier precedence system via Viper:

1. Command-line flags (highest priority)
2. Environment variables
3. Default values

**Key Parameters:**

- `--certname` / `CERTNAME` - Certificate name (required)
- `--puppet-server` / `PUPPET_SERVER` - Puppet server hostname
- `--debug` / `DEBUG` - Enable debug logging
- `--no-reboot` / `NO_REBOOT` - Disable automatic reboot after updates
- `--skip-openvox` / `SKIP_OPENVOX` - Skip Puppet agent run

## Dependencies

- Cobra for CLI framework
- Viper for configuration management
- gRPC for remote logging
- Bitfield/script for shell command execution
