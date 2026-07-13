{
  description = "Prometheus exporter for Telekom Glasfaser-Modem 2";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, utils }:
    let
      statusJson = ''
        [{"vartype":"value","varid":"device_name","varvalue":"Glasfaser-Modem 2"},{"vartype":"value","varid":"rebooting","varvalue":"0"},{"vartype":"value","varid":"ploam_state","varvalue":""},{"vartype":"value","varid":"ploam_success","varvalue":"1"},{"vartype":"value","varid":"save_fails","varvalue":"0"},{"vartype":"value","varid":"service_mode","varvalue":"0"},{"vartype":"page_title","varid":"title","varvalue":"Glasfaser-Modem 2 Konfigurationsprogramm"},{"vartype":"value","varid":"datetime","varvalue":"12.07.2026 20:35:24"},{"vartype":"value","varid":"firmware_version","varvalue":"090144.1.0.009"},{"vartype":"value","varid":"hardware_revision","varvalue":"V1"},{"vartype":"value","varid":"fw_version_standby","varvalue":"090144.1.0.006"},{"vartype":"value","varid":"hardware_state","varvalue":"1"},{"vartype":"value","varid":"txpackets","varvalue":"-900897767"},{"vartype":"value","varid":"txbytes","varvalue":"3473000885047"},{"vartype":"value","varid":"rxpackets","varvalue":"1572888595"},{"vartype":"value","varid":"rxbytes","varvalue":"577922114376"},{"vartype":"value","varid":"rxdrop_packets","varvalue":"0"},{"vartype":"value","varid":"link_status","varvalue":"0"},{"vartype":"value","varid":"stability","varvalue":"19849619"},{"vartype":"value","varid":"rxbip_crc","varvalue":"0"},{"vartype":"value","varid":"serial_number","varvalue":"2343DAAR123DAS08"},{"vartype":"value","varid":"txpower","varvalue":"2.38"},{"vartype":"value","varid":"rxpower","varvalue":"-15.62"},{"vartype":"value","varid":"ui_version","varvalue":"2.18.161"}]
      '';

      fwJson = ''
        [{"vartype":"value","varid":"device_name","varvalue":"Glasfaser-Modem 2"},{"vartype":"value","varid":"rebooting","varvalue":"0"},{"vartype":"value","varid":"save_fails","varvalue":"0"},{"vartype":"value","varid":"service_mode","varvalue":"0"},{"vartype":"page_title","varid":"title","varvalue":"Glasfaser-Modem 2 Konfigurationsprogramm"},{"vartype":"value","varid":"autofw_active","varvalue":"1"},{"vartype":"value","varid":"firmware_version","varvalue":"090144.1.0.009"},{"vartype":"value","varid":"firmware_date","varvalue":"2025-05-23 05:02:21"}]
      '';
    in
    utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      rec {
        packages.default = pkgs.buildGoModule {
          pname = "telekom-gfmodem2-exporter";
          version = "1.0.0";

          src = ./.;

          vendorHash = null;
        };

        apps.default = {
          type = "app";
          program = "${packages.default}/bin/telekom-gfmodem2-exporter";
          meta = {
            description = "Prometheus exporter for Telekom Glasfaser-Modem 2";
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [ go nixpkgs-fmt ];
        };

        checks.test = pkgs.testers.runNixOSTest {
          name = "telekom-gfmodem2-exporter-test";
          nodes = {
            machine = { config, lib, pkgs, ... }: {
              imports = [ self.nixosModules.default ];
              
              services.telekom-gfmodem2-exporter = {
                enable = true;
                modemAddress = "127.0.0.1:8080";
                listenAddress = "127.0.0.1:9877";
              };

              systemd.services.mock-modem = {
                description = "Mock Telekom Glasfaser-Modem 2 HTTP server";
                wantedBy = [ "multi-user.target" ];
                serviceConfig = {
                  ExecStart = let
                    mockScript = pkgs.writeText "mock-modem.py" ''
                      import http.server
                      import socketserver
                      import json

                      PORT = 8080
                      STATUS_DATA = json.loads(r"""${builtins.toJSON (builtins.fromJSON statusJson)}""")
                      FW_DATA = json.loads(r"""${builtins.toJSON (builtins.fromJSON fwJson)}""")

                      class MockHandler(http.server.BaseHTTPRequestHandler):
                          protocol_version = 'HTTP/1.0'
                          def do_GET(self):
                              if self.path == '/ONT/client/data/Status.json':
                                  self.send_response(200)
                                  self.send_header('Content-type', 'application/javascript')
                                  self.end_headers()
                                  self.wfile.write(json.dumps(STATUS_DATA).encode())
                              elif self.path == '/ONT/client/data/FirmwareUpdate.json':
                                  self.send_response(200)
                                  self.send_header('Content-type', 'application/javascript')
                                  self.end_headers()
                                  self.wfile.write(json.dumps(FW_DATA).encode())
                              else:
                                  self.send_response(404)
                                  self.end_headers()

                      with socketserver.TCPServer(("", PORT), MockHandler) as httpd:
                          print("serving at port", PORT)
                          httpd.serve_forever()
                    '';
                  in "${pkgs.python3}/bin/python3 ${mockScript}";
                  DynamicUser = true;
                };
              };
            };
          };

          testScript = ''
            machine.wait_for_unit("mock-modem.service")
            machine.wait_for_unit("telekom-gfmodem2-exporter.service")
            machine.wait_for_open_port(8080)
            machine.wait_for_open_port(9877)

            metrics = machine.succeed("curl -s http://127.0.0.1:9877/metrics")
            
            if 'glasfaser_modem_up 1' not in metrics:
                raise Exception("Exporter did not report modem as up (glasfaser_modem_up 1)")

            expected_metrics = [
                'glasfaser_rxpower -15.62',
                'glasfaser_txpower 2.38',
                'glasfaser_stability 1.9849619e+07',
                'glasfaser_txbytes 3.473000885047e+12',
                'glasfaser_rxbytes 5.77922114376e+11',
                'glasfaser_rebooting 0',
                'glasfaser_autofw_active 1',
            ]
            for m in expected_metrics:
                if m not in metrics:
                    raise Exception(f"Expected metric '{m}' was not found. Metrics returned:\n{metrics}")

            expected_labels = 'glasfaser_modem_info{datetime="12.07.2026 20:35:24",device_name="Glasfaser-Modem 2",firmware_date="2025-05-23 05:02:21",firmware_version="090144.1.0.009",fw_version_standby="090144.1.0.006",hardware_revision="V1",ploam_state="",serial_number="2343DAAR123DAS08",ui_version="2.18.161"} 1'
            if expected_labels not in metrics:
                raise Exception(f"Expected info labels not found. Metrics returned:\n{metrics}")

            print("All tests passed successfully!")
          '';
        };
      }
    ) // {
      nixosModules.default = { config, lib, pkgs, ... }:
        let
          cfg = config.services.telekom-gfmodem2-exporter;
        in
        {
          options.services.telekom-gfmodem2-exporter = {
            enable = lib.mkEnableOption "Telekom Glasfaser-Modem 2 Exporter";

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
              description = "The package to use for the exporter.";
            };

            modemAddress = lib.mkOption {
              type = lib.types.str;
              default = "192.168.100.1";
              description = "Host or IP address of the modem.";
            };

            listenAddress = lib.mkOption {
              type = lib.types.str;
              default = ":9877";
              description = "Address to listen on for web interface and telemetry.";
            };

            logLevel = lib.mkOption {
              type = lib.types.enum [ "debug" "info" "warn" "error" ];
              default = "info";
              description = "Logging level.";
            };

            openFirewall = lib.mkOption {
              type = lib.types.bool;
              default = false;
              description = "Open port in the firewall.";
            };
          };

          config = lib.mkIf cfg.enable {
            systemd.services.telekom-gfmodem2-exporter = {
              description = "Telekom Glasfaser-Modem 2 Exporter";
              after = [ "network.target" ];
              wantedBy = [ "multi-user.target" ];

              serviceConfig = {
                ExecStart = "${cfg.package}/bin/telekom-gfmodem2-exporter "
                  + "-modem-address ${lib.escapeShellArg cfg.modemAddress} "
                  + "-listen-address ${lib.escapeShellArg cfg.listenAddress} "
                  + "-log-level ${lib.escapeShellArg cfg.logLevel}";
                
                Restart = "always";
                DynamicUser = true;
                ProtectHome = true;
                ProtectSystem = "full";
                NoNewPrivileges = true;
              };
            };

            networking.firewall.allowedTCPPorts = lib.mkIf cfg.openFirewall [
              (lib.toInt (lib.last (lib.splitString ":" cfg.listenAddress)))
            ];
          };
        };
    };
}
