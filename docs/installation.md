# Installation Guide

This document explains the steps to install and run prometheus-slurm-sd.

## Installation from Binary

1. Download the latest release binary from the [GitHub Releases](https://github.com/yuuki/prometheus-slurm-sd/releases) page.
2. Extract the downloaded archive.
3. Place the binary in an executable path.

```bash
tar xzf prometheus-slurm-sd_X.Y.Z_linux_amd64.tar.gz
sudo mv prometheus-slurm-sd /usr/local/bin/
```

## Building from Source

### Prerequisites

- Go 1.24 or later
- Git

### Build Steps

```bash
# Clone the repository
git clone https://github.com/yuuki/prometheus-slurm-sd.git
cd prometheus-slurm-sd

# Build
go build

# Install (optional)
sudo mv prometheus-slurm-sd /usr/local/bin/
```

## Running with Docker

```bash
docker run -p 8080:8080 -v /path/to/config.yaml:/app/config.yaml yuuki/prometheus-slurm-sd:latest
```

## Setting up as a System Service (systemd)

Example configuration for running as a systemd service:

1. Create a service definition file:

```bash
sudo cat > /etc/systemd/system/prometheus-slurm-sd.service << EOF
[Unit]
Description=Prometheus Service Discovery for Slurm
After=network.target

[Service]
User=prometheus
ExecStart=/usr/local/bin/prometheus-slurm-sd --config.file=/etc/prometheus/slurm-sd.yaml
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF
```

2. Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable prometheus-slurm-sd
sudo systemctl start prometheus-slurm-sd
```

3. Check the service status:

```bash
sudo systemctl status prometheus-slurm-sd
```
