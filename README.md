# Docker Info (di)

Docker info is a small utility that emulates `docker ps`, but with better grouping,
cleaner port information, and a nicer layout. I manage my homelab with multiple
`docker compose` stacks, and I wanted something nicer to get the status of
all docker containers.

## Features

- **Project Grouping**: Automatically groups containers by their Docker Compose project.
- **Smart Sizing**: Detects terminal width and intelligently trims long names/images to prevent ugly wrapping.
- **Color-Coded Status**: Instantly spot `Up`, `Exited`, or `Restarting` containers.
- **Clean Ports**: Port information is hidden by default and simplified when shown (compacts redundant IPv4/IPv6 mappings).
- **Fast**: Built in Go using the official Docker SDK for direct `docker.sock` communication.

## Usage

```bash
# Basic view
./docker-info

# Show port mappings
./docker-info -p
```

## Installation

### Prerequisites
- Docker installed and running.
- Go 1.26+ (if building from source).

### Building
```bash
make build
```

### System-wide Install
```bash
sudo make install
```

## Configuration
The utility should respect standard Docker environment variables (e.g., `DOCKER_HOST`).
