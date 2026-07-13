package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Noodlesalat/telekom-gfmodem2-exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var (
		modemAddressFlag      string
		modemAddressAliasFlag string
		listenAddress         string
		logLevelStr           string
		timeout               time.Duration
	)

	flag.StringVar(&modemAddressFlag, "modem-address", "", "Host or IP address of the Telekom Glasfaser-Modem 2")
	flag.StringVar(&modemAddressAliasFlag, "modem", "", "Alias for -modem-address")
	flag.StringVar(&listenAddress, "listen-address", ":9877", "Address to listen on for web interface and telemetry")
	flag.StringVar(&logLevelStr, "log-level", "info", "Only log messages with the given severity or above. One of: debug, info, warn, error")
	flag.DurationVar(&timeout, "timeout", 5*time.Second, "Timeout for scraping the modem")
	flag.Parse()

	modemAddress := "192.168.100.1"
	if modemAddressFlag != "" {
		modemAddress = modemAddressFlag
	} else if modemAddressAliasFlag != "" {
		modemAddress = modemAddressAliasFlag
	}

	var level slog.Level
	switch logLevelStr {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	slog.Info("Starting telekom-gfmodem2-exporter")
	slog.Info("Modem address configured", "address", modemAddress)
	slog.Info("Listen address configured", "address", listenAddress)

	collector := exporter.NewModemCollector(modemAddress, timeout)
	prometheus.MustRegister(collector)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
			<head><title>Telekom Glasfaser-Modem 2 Exporter</title></head>
			<body>
			<h1>Telekom Glasfaser-Modem 2 Exporter</h1>
			<p><a href="/metrics">Metrics</a></p>
			</body>
			</html>`))
	})

	slog.Info("Listening on", "address", listenAddress)
	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		slog.Error("Failed to listen and serve", "error", err)
		os.Exit(1)
	}
}
