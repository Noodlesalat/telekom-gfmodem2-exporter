package exporter

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestModemCollector(t *testing.T) {
	statusJSON := `[
		{"vartype":"value","varid":"rxpower","varvalue":"-15.62"},
		{"vartype":"value","varid":"txpower","varvalue":"2.38"},
		{"vartype":"value","varid":"stability","varvalue":"19849619"},
		{"vartype":"value","varid":"rebooting","varvalue":"0"}
	]`

	fwJSON := `[
		{"vartype":"value","varid":"autofw_active","varvalue":"1"},
		{"vartype":"value","varid":"device_name","varvalue":"Glasfaser-Modem 2"}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		if r.URL.Path == "/ONT/client/data/Status.json" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(statusJSON))
		} else if r.URL.Path == "/ONT/client/data/FirmwareUpdate.json" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fwJSON))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	collector := NewModemCollector(server.URL, 2*time.Second)

	reg := prometheus.NewRegistry()
	if err := reg.Register(collector); err != nil {
		t.Fatalf("failed to register collector: %v", err)
	}

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	foundMetrics := make(map[string]float64)
	for _, mf := range mfs {
		for _, metric := range mf.Metric {
			if metric.Gauge != nil {
				foundMetrics[mf.GetName()] = metric.Gauge.GetValue()
			}
		}
	}

	if foundMetrics["glasfaser_modem_up"] != 1 {
		t.Errorf("expected glasfaser_modem_up to be 1, got %v", foundMetrics["glasfaser_modem_up"])
	}
	if foundMetrics["glasfaser_rxpower"] != -15.62 {
		t.Errorf("expected glasfaser_rxpower to be -15.62, got %v", foundMetrics["glasfaser_rxpower"])
	}
	if foundMetrics["glasfaser_txpower"] != 2.38 {
		t.Errorf("expected glasfaser_txpower to be 2.38, got %v", foundMetrics["glasfaser_txpower"])
	}
	if foundMetrics["glasfaser_stability"] != 19849619 {
		t.Errorf("expected glasfaser_stability to be 19849619, got %v", foundMetrics["glasfaser_stability"])
	}
	if foundMetrics["glasfaser_autofw_active"] != 1 {
		t.Errorf("expected glasfaser_autofw_active to be 1, got %v", foundMetrics["glasfaser_autofw_active"])
	}
	if foundMetrics["glasfaser_rebooting"] != 0 {
		t.Errorf("expected glasfaser_rebooting to be 0, got %v", foundMetrics["glasfaser_rebooting"])
	}
}
