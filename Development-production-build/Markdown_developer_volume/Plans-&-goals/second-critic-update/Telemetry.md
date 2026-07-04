# Arena Telemetry & Observability Guide

## 🛰️ Overview
This document serves as the technical specification for the Observability Kernel, which is a core component of the Authoritative Blueprint.

The Arena utilizes a dedicated **Observability Kernel** implemented in `economy_telemetry.go`. This system exports real-time performance and financial data using the Prometheus client library, allowing infrastructure administrators to monitor connection health, ledger integrity, and bootstrap performance via Grafana dashboards.

## 🔌 Connection & Endpoints
*   **Metrics Port**: `9090` (Isolated management port)
*   **Endpoint**: `http://localhost:9090/metrics`
*   **Format**: Standard OpenMetrics / Prometheus text-based scrape vector.

## 📊 Core Metric Definitions

### 1. System Health & Boot
| Metric Name | Type | Description |
| :--- | :--- | :--- |
| `arena_bootstrap_duration_seconds` | Gauge | The total time (seconds) required to hydrate the authoritative state from blockchain snapshots and local caches during server initialization. |
| `arena_financial_health_status` | Gauge | Binary status of the ledger circularity. **1** = Healthy (Zero Drift), **0** = Compromised (Precision Loss Detected). |

### 2. Economic Velocity (Industrial Loop)
| Metric Name | Type | Description |
| :--- | :--- | :--- |
| `arena_economy_inflow_total_micro_vbv` | Counter | Cumulative sum of all system taxes and fees vetted by the `TokenSinkRouter`. |
| `arena_economy_outflow_total_micro_vbv` | Counter | Cumulative sum of all distributed revenue (Faucet, Clubs, Governors). |
| `arena_economy_net_drift_micro_vbv` | Gauge | The current variance between Inflow and Outflow. Must strictly remain at **0** to maintain the Industrial Seal. |
| `arena_governor_taxes_micro_vbv` | GaugeVec | Real-time tracking of localized tax policies per territory. |

## 🛠️ Infrastructure Configuration

### Prometheus Scrape Config (`prometheus.yml`)
```yaml
scrape_configs:
  - job_name: 'virtualbabes_arena_engine'
    scrape_interval: 10s
    static_configs:
      - targets: ['localhost:9090']
```

### Grafana Visualization Strategy
To assist with monitoring the live environment, use the following PromQL queries for your dashboard panels:

1.  **System Uptime & Solvency (Stat Gauge)**:
    *   Query: `arena_financial_health_status` (Current status invariant. 1 = Healthy, 0 = Drifting)
    *   Mapping: `1` -> "OPERATIONAL" (Green), `0` -> "CRITICAL" (Red).
2.  **Solvency Ratio (Physical vs. Virtual)**:
    *   Query: `arena_economy_inflow_total_micro_vbv / arena_economy_outflow_total_micro_vbv`
    *   Mapping: Targets >= 1.0.

2.  **Token Velocity (Time Series)**:
    *   Query: `rate(arena_economy_inflow_total_micro_vbv[1m])`
    *   Utility: Visualizes real-time transaction volume and economic activity.
3.  **Precision Leak Monitor (Singlestat)**:
    *   Query: `arena_economy_net_drift_micro_vbv`
    *   Alerting: Trigger a PagerDuty/Discord alert if this value deviates from **0**.

## 🔍 Troubleshooting Connections
If the telemetry stream is unresponsive:
1.  **Check Management Port**: Ensure port `9090` is not blocked by the host firewall or Render's service configuration.
2.  **Verify Context**: The telemetry server is wired to the global server context. If the main server loop panics, the telemetry server will shut down automatically after 5 seconds to clear the port.
3.  **Audit Logs**: Check `admin_audit.log` for `TELEMETRY_CRASH` entries if the exporter fails to bind to the port.

---
**PILLAR 4**: System Observability & Telemetry Finalization.
**Dual-Target Build**: Supported on Server (Linux/Docker) targets only.