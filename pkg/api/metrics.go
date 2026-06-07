package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MetricsCollector implements metrics collection for CubeVE
type MetricsCollector struct {
	incus      *IncusClient
	startTime  time.Time
	counters   map[string]int64
	gauges     map[string]float64
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(incus *IncusClient) *MetricsCollector {
	return &MetricsCollector{
		incus:     incus,
		startTime: time.Now(),
		counters:  make(map[string]int64),
		gauges:    make(map[string]float64),
	}
}

// CollectInstanceMetrics collects instance metrics
func (c *MetricsCollector) CollectInstanceMetrics(ctx context.Context) (*InstanceMetrics, error) {
	instances, err := c.incus.ListInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	metrics := &InstanceMetrics{
		Total:     int64(len(instances)),
		ByState:   make(map[string]int64),
		ByRuntime: make(map[string]int64),
	}

	for _, name := range instances {
		inst, err := c.incus.GetInstance(ctx, name)
		if err != nil {
			continue
		}

		metrics.ByState[inst.Status]++
		metrics.ByRuntime["incus"]++
	}

	return metrics, nil
}

// CollectStorageMetrics collects storage metrics
func (c *MetricsCollector) CollectStorageMetrics(ctx context.Context) (*StorageMetrics, error) {
	pools, err := c.incus.ListStoragePools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage pools: %w", err)
	}

	metrics := &StorageMetrics{
		TotalPools: int64(len(pools)),
		Pools:      make(map[string]PoolMetrics),
	}

	for _, name := range pools {
		pool, err := c.incus.GetStoragePool(ctx, name)
		if err != nil {
			continue
		}

		metrics.Pools[name] = PoolMetrics{
			Driver: pool.Driver,
		}
	}

	return metrics, nil
}

// CollectNetworkMetrics collects network metrics
func (c *MetricsCollector) CollectNetworkMetrics(ctx context.Context) (*NetworkMetrics, error) {
	networks, err := c.incus.ListNetworks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	metrics := &NetworkMetrics{
		TotalNetworks: int64(len(networks)),
		Networks:      make(map[string]NetMetrics),
	}

	for _, name := range networks {
		net, err := c.incus.GetNetwork(ctx, name)
		if err != nil {
			continue
		}

		metrics.Networks[name] = NetMetrics{
			Type: net.Type,
		}
	}

	return metrics, nil
}

// CollectSystemMetrics collects system-wide metrics
func (c *MetricsCollector) CollectSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	info, err := c.incus.GetServerInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}

	metrics := &SystemMetrics{
		APIVersion:   info.APIVersion,
		Uptime:       time.Since(c.startTime).String(),
		StartTime:    c.startTime.Format(time.RFC3339),
	}

	if env, ok := info.Environment["kernel_version"]; ok {
		metrics.KernelVersion = fmt.Sprintf("%v", env)
	}

	if env, ok := info.Environment["server_version"]; ok {
		metrics.ServerVersion = fmt.Sprintf("%v", env)
	}

	return metrics, nil
}

// GetAllMetrics returns all metrics combined
func (c *MetricsCollector) GetAllMetrics(ctx context.Context) (*AllMetrics, error) {
	instances, err := c.CollectInstanceMetrics(ctx)
	if err != nil {
		instances = &InstanceMetrics{}
	}

	storage, err := c.CollectStorageMetrics(ctx)
	if err != nil {
		storage = &StorageMetrics{}
	}

	networks, err := c.CollectNetworkMetrics(ctx)
	if err != nil {
		networks = &NetworkMetrics{}
	}

	system, err := c.CollectSystemMetrics(ctx)
	if err != nil {
		system = &SystemMetrics{}
	}

	return &AllMetrics{
		Timestamp: time.Now().Format(time.RFC3339),
		Instances: instances,
		Storage:   storage,
		Network:   networks,
		System:    system,
	}, nil
}

// Metrics types

type InstanceMetrics struct {
	Total     int64            `json:"total"`
	ByState   map[string]int64 `json:"by_state"`
	ByRuntime map[string]int64 `json:"by_runtime"`
}

type StorageMetrics struct {
	TotalPools int64                `json:"total_pools"`
	Pools      map[string]PoolMetrics `json:"pools"`
}

type PoolMetrics struct {
	Driver string `json:"driver"`
	Size   string `json:"size,omitempty"`
	Used   string `json:"used,omitempty"`
	Free   string `json:"free,omitempty"`
}

type NetworkMetrics struct {
	TotalNetworks int64               `json:"total_networks"`
	Networks      map[string]NetMetrics `json:"networks"`
}

type NetMetrics struct {
	Type      string `json:"type"`
	Addresses string `json:"addresses,omitempty"`
}

type SystemMetrics struct {
	APIVersion    string `json:"api_version"`
	ServerVersion string `json:"server_version,omitempty"`
	KernelVersion string `json:"kernel_version,omitempty"`
	Uptime        string `json:"uptime"`
	StartTime     string `json:"start_time"`
}

type AllMetrics struct {
	Timestamp string           `json:"timestamp"`
	Instances *InstanceMetrics `json:"instances,omitempty"`
	Storage   *StorageMetrics  `json:"storage,omitempty"`
	Network   *NetworkMetrics  `json:"network,omitempty"`
	System    *SystemMetrics   `json:"system,omitempty"`
}

// MetricsHandler returns a HTTP handler for metrics
func MetricsHandler(collector *MetricsCollector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		metrics, err := collector.GetAllMetrics(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	}
}

// PrometheusMetricsHandler returns a HTTP handler for Prometheus metrics
func PrometheusMetricsHandler(collector *MetricsCollector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		metrics, err := collector.GetAllMetrics(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)

		// Write Prometheus format
		fmt.Fprintf(w, "# HELP cubeve_instances_total Total number of instances\n")
		fmt.Fprintf(w, "# TYPE cubeve_instances_total gauge\n")
		fmt.Fprintf(w, "cubeve_instances_total %d\n", metrics.Instances.Total)

		fmt.Fprintf(w, "# HELP cubeve_instances_by_state Number of instances by state\n")
		fmt.Fprintf(w, "# TYPE cubeve_instances_by_state gauge\n")
		for state, count := range metrics.Instances.ByState {
			fmt.Fprintf(w, "cubeve_instances_by_state{state=\"%s\"} %d\n", state, count)
		}

		fmt.Fprintf(w, "# HELP cubeve_storage_pools_total Total number of storage pools\n")
		fmt.Fprintf(w, "# TYPE cubeve_storage_pools_total gauge\n")
		fmt.Fprintf(w, "cubeve_storage_pools_total %d\n", metrics.Storage.TotalPools)

		fmt.Fprintf(w, "# HELP cubeve_networks_total Total number of networks\n")
		fmt.Fprintf(w, "# TYPE cubeve_networks_total gauge\n")
		fmt.Fprintf(w, "cubeve_networks_total %d\n", metrics.Network.TotalNetworks)

		fmt.Fprintf(w, "# HELP cubeve_api_info API information\n")
		fmt.Fprintf(w, "# TYPE cubeve_api_info gauge\n")
		fmt.Fprintf(w, "cubeve_api_info{version=\"%s\"} 1\n", metrics.System.APIVersion)
	}
}