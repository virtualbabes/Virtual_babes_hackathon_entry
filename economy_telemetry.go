//go:build !js && !wasm

package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// TelemetryLogger manages the export of internal engine metrics to Prometheus.
// PILLAR 4: System Observability.
type TelemetryLogger struct {
	Mu             sync.Mutex
	HttpPort       string
	ServerInstance *http.Server

	// Prometheus Time-Series Vector Definitions
	BootDurationGauge prometheus.Gauge
	SystemStatusGauge prometheus.Gauge
	InflowCounter     prometheus.Counter
	OutflowCounter    prometheus.Counter
	NetDriftGauge     prometheus.Gauge
	PlatformFeeCounter prometheus.Counter // PILLAR 2: Enforcement tracking
	GhostReclaimCounter prometheus.Counter
	StagnationFeeCounter prometheus.Counter
	GovernorTaxGauge  *prometheus.GaugeVec // PILLAR 4: Regional Telemetry
}

// NewTelemetryLogger initializes the metrics registry and definitions.
func NewTelemetryLogger(port string) *TelemetryLogger {
	tl := &TelemetryLogger{
		HttpPort: port,

		BootDurationGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "arena_bootstrap_duration_seconds",
			Help: "The absolute time spent reading and hydrating the authoritative save state from disk.",
		}),
		SystemStatusGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "arena_financial_health_status",
			Help: "Current health status invariant of the ledger. 1 = Clean Operational, 0 = Compromised Drifting.",
		}),
		InflowCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "arena_economy_inflow_total_micro_vbv",
			Help: "Cumulative transactional taxes vetted and passed through the Token-Sink system router.",
		}),
		OutflowCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "arena_economy_outflow_total_micro_vbv",
			Help: "Cumulative distributed revenue values pushed out to Faucets, Clubs, and Governance nodes.",
		}),
		NetDriftGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "arena_economy_net_drift_micro_vbv",
			Help: "The current structural precision variance. Must strictly remain at 0.",
		}),
		PlatformFeeCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "arena_enforcement_platform_fees_total_micro_vbv",
			Help: "Cumulative surcharges collected from self-redemption events.",
		}),
		GhostReclaimCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "arena_enforcement_ghost_reclaimed_total_micro_vbv",
			Help: "Cumulative revenue recycled from uninitialized creators.",
		}),
		StagnationFeeCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "arena_enforcement_stagnation_fees_total_micro_vbv",
			Help: "Cumulative fees collected from stagnant accounts.",
		}),
		GovernorTaxGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "arena_governor_taxes_micro_vbv",
			Help: "Cumulative taxes collected by Regional Governors per district.",
		}, []string{"district"}),
	}

	// Register variables securely within the global Prometheus collector pool
	prometheus.MustRegister(tl.BootDurationGauge)
	prometheus.MustRegister(tl.SystemStatusGauge)
	prometheus.MustRegister(tl.InflowCounter)
	prometheus.MustRegister(tl.OutflowCounter)
	prometheus.MustRegister(tl.PlatformFeeCounter)
	prometheus.MustRegister(tl.GhostReclaimCounter)
	prometheus.MustRegister(tl.StagnationFeeCounter)
	prometheus.MustRegister(tl.NetDriftGauge)
	prometheus.MustRegister(tl.GovernorTaxGauge)

	return tl
}

// StartTelemetryServer opens up a non-blocking background HTTP daemon on an isolated management port.
func (tl *TelemetryLogger) StartTelemetryServer(ctx context.Context) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler()) // Exposes standard Prometheus scrape vector string

	tl.ServerInstance = &http.Server{
		Addr:    ":" + tl.HttpPort,
		Handler: mux,
	}

	go func() {
		fmt.Printf(" [Telemetry] Exporter active. Serving metrics stream out at http://localhost:%s/metrics\n", tl.HttpPort)
		if err := tl.ServerInstance.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf(" [Telemetry Error] Exporter crashed unexpectedly: %v\n", err)
		}
	}()

	// Listen for top-level system interrupt commands to shut down cleanly
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tl.ServerInstance.Shutdown(shutdownCtx)
		fmt.Println(" [Telemetry] Exporter daemon stopped successfully.")
	}()
}

// RecordBootstrapMetrics calculates and pushes the boot initialization profile vectors.
func (tl *TelemetryLogger) RecordBootstrapMetrics(startTime time.Time, isSuccessful bool) {
	duration := time.Since(startTime).Seconds()
	tl.BootDurationGauge.Set(duration)

	if isSuccessful {
		tl.SystemStatusGauge.Set(1)
	} else {
		tl.SystemStatusGauge.Set(0)
	}
}

// UpdateRealTimeAccounting vectors metrics from the Audit Reporter to Prometheus variables.
// PILLAR 2: Ledger Integrity.
func (tl *TelemetryLogger) UpdateRealTimeAccounting(inflowDelta uint64, outflowDelta uint64, absoluteDrift int64) {
	tl.InflowCounter.Add(float64(inflowDelta))
	tl.OutflowCounter.Add(float64(outflowDelta))
	tl.NetDriftGauge.Set(float64(absoluteDrift))
}

// UpdateEnforcementAccounting vectors metrics from the Audit Reporter to Prometheus variables.
// PILLAR 2: Ledger Integrity.
func (tl *TelemetryLogger) UpdateEnforcementAccounting(platform uint64, ghost uint64, stagnation uint64) {
	if platform > 0 { tl.PlatformFeeCounter.Add(float64(platform)) }
	if ghost > 0 { tl.GhostReclaimCounter.Add(float64(ghost)) }
	if stagnation > 0 { tl.StagnationFeeCounter.Add(float64(stagnation)) }
}

// UpdateGovernorTax pushes the latest cumulative tax totals for a specific district.
// PILLAR 1: Political Influence.
func (tl *TelemetryLogger) UpdateGovernorTax(district string, amount uint64) {
	// PILLAR 2: Integer delta accumulation.
	tl.GovernorTaxGauge.WithLabelValues(district).Add(float64(amount))
}
