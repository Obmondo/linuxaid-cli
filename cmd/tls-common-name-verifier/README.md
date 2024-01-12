# TLS Certificate Common Name Verifier

This Go program connects to specified servers using their IP addresses and verifies if the presented TLS certificate's Common Name matches the expected value. The configuration is read from a YAML file.

## Getting Started

### Prerequisites

- Go (Golang) installed on your machine.

### Installation

1. Clone the repository:

    ```bash
    git clone ssh://git@gitea.obmondo.com:2223/EnableIT/go-scripts.git
    cd go-scripts/cmd/datacenters
    ```

2. Build the Go program:

    ```bash
    make build
    ```

### Usage

1. Create a YAML configuration file (`config.yaml`) with the server details:

    ```yaml
    domains:
      - ip: 142.250.206.174
        common_name: google.com
      - ip: 20.112.250.133
        common_name: microsoft.com
    ```

2. Run the program:

    ```bash
    ./tls-common-name-verifier
    ```

   The program will connect to each specified server and verify if the Common Name in the TLS certificate matches the expected value.

### Configuration

- The YAML configuration file (`config.yaml`) should be structured as follows:

  ```yaml
  domains:
    - ip: <server_ip>
      common_name: <expected_common_name>
    - ip: <another_server_ip>
      common_name: <another_expected_common_name>
