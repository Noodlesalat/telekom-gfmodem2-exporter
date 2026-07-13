# Telekom Glasfaser-Modem 2 Prometheus Exporter

A Prometheus exporter for the Telekom Glasfaser-Modem 2 (fiber optic modem) written in Go. It queries the modem's JSON status API directly and exposes the data as Prometheus-compatible metrics.

## Features

- Written in Go, producing a single static binary.
- On-demand scraping (directly queries the modem on scraping requests, adhering to Prometheus timeouts).
- Configurable listen address, log level, and modem target.
- Ready-to-use NixOS Flake module & integration test.
- Pre-configured Grafana dashboard template.
- Built for Linux, including MIPS targets for direct execution on OpenWRT.

## Metrics

The exporter exposes the following metrics:

| Metric Name | Type | Description |
|---|---|---|
| `glasfaser_modem_up` | Gauge | Indicates if the modem is reachable and responding (1 = up, 0 = down) |
| `glasfaser_rebooting` | Gauge | Rebooting status (1 = rebooting, 0 = normal) |
| `glasfaser_ploam_success` | Gauge | PLOAM success status |
| `glasfaser_save_fails` | Gauge | Configuration save fails count |
| `glasfaser_service_mode` | Gauge | Service mode status |
| `glasfaser_hardware_state` | Gauge | Hardware state |
| `glasfaser_txpackets` | Gauge | Transmitted packets |
| `glasfaser_txbytes` | Gauge | Transmitted bytes |
| `glasfaser_rxpackets` | Gauge | Received packets |
| `glasfaser_rxbytes` | Gauge | Received bytes |
| `glasfaser_rxdrop_packets` | Gauge | Dropped received packets |
| `glasfaser_link_status` | Gauge | Link status (0 = up, other = down) |
| `glasfaser_stability` | Gauge | Stability value (typically uptime in seconds) |
| `glasfaser_rxbip_crc` | Gauge | Received CRC errors (BIP) |
| `glasfaser_txpower` | Gauge | TX optical power (dBm) |
| `glasfaser_rxpower` | Gauge | RX optical power (dBm) |
| `glasfaser_autofw_active` | Gauge | Auto firmware update active status |
| `glasfaser_modem_info` | Gauge | Static metadata about the modem (labels: `device_name`, `serial_number`, `firmware_version`, `firmware_date`, `hardware_revision`, `fw_version_standby`, `ui_version`, `ploam_state`, `datetime`) |

## Installation & Usage

### Running the Binary

Download the compiled binary for your architecture from the [Releases](https://github.com/Noodlesalat/telekom-gfmodem2-exporter/releases) page or compile it from source.

```bash
./telekom-gfmodem2-exporter -modem-address 192.168.100.1 -listen-address :9877
```

### CLI Flags

- `-modem-address` (or `-modem`): Host or IP address of the Telekom Glasfaser-Modem 2 (default `192.168.100.1`).
- `-listen-address`: Address to listen on for web interface and telemetry (default `:9877`).
- `-log-level`: Only log messages with the given severity or above. Options: `debug`, `info`, `warn`, `error` (default `info`).
- `-timeout`: Timeout for scraping requests to the modem (default `5s`).

### Building from Source

Ensure you have Go installed (version 1.22 or newer).

```bash
git clone https://github.com/Noodlesalat/telekom-gfmodem2-exporter.git
cd telekom-gfmodem2-exporter
go build -o telekom-gfmodem2-exporter
```

---

## NixOS Configuration (Flake)

This repository is a Nix Flake. You can run the exporter or include the NixOS module directly.

### Running with Nix

```bash
nix run github:Noodlesalat/telekom-gfmodem2-exporter -- -modem-address 192.168.100.1
```

### Adding to NixOS Configuration

Include the flake input and import the module:

```nix
{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    telekom-gfmodem2-exporter.url = "github:Noodlesalat/telekom-gfmodem2-exporter";
  };

  outputs = { self, nixpkgs, telekom-gfmodem2-exporter, ... }: {
    nixosConfigurations.myrouter = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        telekom-gfmodem2-exporter.nixosModules.default
        ({ pkgs, ... }: {
          services.telekom-gfmodem2-exporter = {
            enable = true;
            modemAddress = "192.168.100.1";
            listenAddress = ":9877";
            openFirewall = true;
          };
        })
      ];
    };
  };
}
```

### NixOS Module Options

- `services.telekom-gfmodem2-exporter.enable`: Whether to enable the exporter service.
- `services.telekom-gfmodem2-exporter.package`: The package to use for the exporter (defaults to flake package).
- `services.telekom-gfmodem2-exporter.modemAddress`: IP address or host of the modem (default: `192.168.100.1`).
- `services.telekom-gfmodem2-exporter.listenAddress`: Listen port/address (default: `:9877`).
- `services.telekom-gfmodem2-exporter.logLevel`: Logging verbosity (default: `info`).
- `services.telekom-gfmodem2-exporter.openFirewall`: Open the port configured in `listenAddress` in the NixOS firewall.

---

## Prometheus Scrape Configuration

Add the exporter as a target in your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'telekom-gfmodem'
    static_configs:
      - targets: ['192.168.100.2:9877'] # Replace with your exporter IP/port
```

## Grafana Dashboard

A template JSON dashboard is provided in [grafana-dashboard.json](grafana-dashboard.json). You can import it directly to Grafana. It includes:
- Modem status indicators.
- Transmitted/Received optical power graph (dBm).
- RX/TX throughput rate (bps).
- Link stability counter.
- Frame / CRC errors panel.
- General hardware & firmware information metadata table.

## License

This project is licensed under the MIT License.
