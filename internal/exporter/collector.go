package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type ModemVariable struct {
	VarType  string `json:"vartype"`
	VarID    string `json:"varid"`
	VarValue string `json:"varvalue"`
}

type ModemCollector struct {
	modemAddress string
	httpClient   *http.Client
	upMetric     *prometheus.Desc
	infoMetric   *prometheus.Desc
	metrics      map[string]*prometheus.Desc
}

func NewModemCollector(modemAddress string, timeout time.Duration) *ModemCollector {
	// Ensure address starts with http:// or https://
	addr := modemAddress
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}

	return &ModemCollector{
		modemAddress: addr,
		httpClient:   &http.Client{Timeout: timeout},
		upMetric: prometheus.NewDesc(
			"glasfaser_modem_up",
			"Indicates if the modem is reachable and responding (1 = up, 0 = down)",
			nil, nil,
		),
		infoMetric: prometheus.NewDesc(
			"glasfaser_modem_info",
			"Static metadata about the modem",
			[]string{
				"device_name",
				"serial_number",
				"firmware_version",
				"firmware_date",
				"hardware_revision",
				"fw_version_standby",
				"ui_version",
				"ploam_state",
				"datetime",
			}, nil,
		),
		metrics: map[string]*prometheus.Desc{
			"rebooting": prometheus.NewDesc(
				"glasfaser_rebooting",
				"Rebooting status",
				nil, nil,
			),
			"ploam_success": prometheus.NewDesc(
				"glasfaser_ploam_success",
				"PLOAM success status",
				nil, nil,
			),
			"save_fails": prometheus.NewDesc(
				"glasfaser_save_fails",
				"Configuration save fails count",
				nil, nil,
			),
			"service_mode": prometheus.NewDesc(
				"glasfaser_service_mode",
				"Service mode status",
				nil, nil,
			),
			"hardware_state": prometheus.NewDesc(
				"glasfaser_hardware_state",
				"Hardware state",
				nil, nil,
			),
			"txpackets": prometheus.NewDesc(
				"glasfaser_txpackets",
				"Transmitted packets",
				nil, nil,
			),
			"txbytes": prometheus.NewDesc(
				"glasfaser_txbytes",
				"Transmitted bytes",
				nil, nil,
			),
			"rxpackets": prometheus.NewDesc(
				"glasfaser_rxpackets",
				"Received packets",
				nil, nil,
			),
			"rxbytes": prometheus.NewDesc(
				"glasfaser_rxbytes",
				"Received bytes",
				nil, nil,
			),
			"rxdrop_packets": prometheus.NewDesc(
				"glasfaser_rxdrop_packets",
				"Dropped received packets",
				nil, nil,
			),
			"link_status": prometheus.NewDesc(
				"glasfaser_link_status",
				"Link status",
				nil, nil,
			),
			"stability": prometheus.NewDesc(
				"glasfaser_stability",
				"Stability value",
				nil, nil,
			),
			"rxbip_crc": prometheus.NewDesc(
				"glasfaser_rxbip_crc",
				"Received CRC errors (BIP)",
				nil, nil,
			),
			"txpower": prometheus.NewDesc(
				"glasfaser_txpower",
				"TX optical power (dBm)",
				nil, nil,
			),
			"rxpower": prometheus.NewDesc(
				"glasfaser_rxpower",
				"RX optical power (dBm)",
				nil, nil,
			),
			"autofw_active": prometheus.NewDesc(
				"glasfaser_autofw_active",
				"Auto firmware update active status",
				nil, nil,
			),
		},
	}
}

func (c *ModemCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.upMetric
	ch <- c.infoMetric
	for _, desc := range c.metrics {
		ch <- desc
	}
}

func (c *ModemCollector) Collect(ch chan<- prometheus.Metric) {
	statusURL := fmt.Sprintf("%s/ONT/client/data/Status.json", c.modemAddress)
	fwURL := fmt.Sprintf("%s/ONT/client/data/FirmwareUpdate.json", c.modemAddress)

	slog.Debug("Scraping modem data", "status_url", statusURL, "fw_url", fwURL)

	var wg sync.WaitGroup
	var statusData, fwData map[string]string
	var errStatus, errFW error

	wg.Add(2)
	go func() {
		defer wg.Done()
		statusData, errStatus = c.fetch(statusURL)
	}()
	go func() {
		defer wg.Done()
		fwData, errFW = c.fetch(fwURL)
	}()
	wg.Wait()

	if errStatus != nil || errFW != nil {
		slog.Error("Failed to scrape modem", "status_err", errStatus, "fw_err", errFW)
		ch <- prometheus.MustNewConstMetric(c.upMetric, prometheus.GaugeValue, 0)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.upMetric, prometheus.GaugeValue, 1)

	combined := make(map[string]string)
	for k, v := range statusData {
		combined[k] = v
	}
	for k, v := range fwData {
		combined[k] = v
	}

	for key, desc := range c.metrics {
		valStr, exists := combined[key]
		var val float64
		if exists && valStr != "--" && valStr != "" {
			var err error
			val, err = strconv.ParseFloat(valStr, 64)
			if err != nil {
				slog.Warn("Failed to parse metric value", "key", key, "value", valStr, "error", err)
				val = 0
			}
		}
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, val)
	}

	getVal := func(key string) string {
		if v, ok := combined[key]; ok {
			return v
		}
		return ""
	}

	ch <- prometheus.MustNewConstMetric(
		c.infoMetric,
		prometheus.GaugeValue,
		1,
		getVal("device_name"),
		getVal("serial_number"),
		getVal("firmware_version"),
		getVal("firmware_date"),
		getVal("hardware_revision"),
		getVal("fw_version_standby"),
		getVal("ui_version"),
		getVal("ploam_state"),
		getVal("datetime"),
	)
}

func (c *ModemCollector) fetch(url string) (map[string]string, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Language", "de,en-US;q=0.7,en;q=0.3")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	var variables []ModemVariable
	if err := json.NewDecoder(resp.Body).Decode(&variables); err != nil {
		return nil, err
	}

	data := make(map[string]string)
	for _, v := range variables {
		data[v.VarID] = v.VarValue
	}
	return data, nil
}
