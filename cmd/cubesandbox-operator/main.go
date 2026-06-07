package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var Version = "dev"

// SandboxPool manages pre-warmed MicroVM instances
type SandboxPool struct {
	size        int
	prewarm     bool
	cowClone    bool
	mu          sync.RWMutex
	instances   []SandboxInstance
	available   chan SandboxInstance
}

// SandboxInstance represents a single MicroVM instance
type SandboxInstance struct {
	ID        string
	Status    string
	VCPUs     int
	MemoryMB  int
	CreatedAt time.Time
	LastUsed  time.Time
}

// SandboxConfig holds the operator configuration
type SandboxConfig struct {
	PoolSize         int
	Prewarm          bool
	CoWClone         bool
	MaxConcurrentBoot int
	DefaultVCPUs     int
	DefaultMemoryMB  int
	KernelPath       string
	RootfsTemplate   string
	NetworkBackend   string
	XDPEnabled       bool
}

func main() {
	var (
		listenAddr   = flag.String("listen", ":8443", "API listen address")
		metricsAddr  = flag.String("metrics", ":9100", "Metrics listen address")
		configPath   = flag.String("config", "/etc/cubesandbox/config.toml", "Config file path")
		poolSize     = flag.Int("pool-size", 2000, "VM pool size")
		prewarm      = flag.Bool("prewarm", true, "Enable prewarming")
	)
	flag.Parse()

	log.Printf("CubeSandbox Operator %s starting...", Version)
	log.Printf("  API: %s", *listenAddr)
	log.Printf("  Metrics: %s", *metricsAddr)
	log.Printf("  Config: %s", *configPath)
	log.Printf("  Pool Size: %d", *poolSize)
	log.Printf("  Prewarm: %v", *prewarm)

	// Load config
	config := &SandboxConfig{
		PoolSize:          *poolSize,
		Prewarm:           *prewarm,
		CoWClone:          true,
		MaxConcurrentBoot: 100,
		DefaultVCPUs:      2,
		DefaultMemoryMB:   512,
		KernelPath:        "/opt/cubesandbox/vmlinux-6.6.0",
		RootfsTemplate:    "/opt/cubesandbox/rootfs-template.img",
		NetworkBackend:    "cube-vs",
		XDPEnabled:        true,
	}

	// Create pool
	pool := NewSandboxPool(config)

	// Start pool manager
	go pool.Manage(context.Background())

	// Start API server
	go startAPIServer(*listenAddr, pool)

	// Start metrics server
	go startMetricsServer(*metricsAddr)

	// Wait for shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down...")
	pool.Shutdown()
	os.Exit(0)
}

// NewSandboxPool creates a new sandbox pool
func NewSandboxPool(config *SandboxConfig) *SandboxPool {
	return &SandboxPool{
		size:      config.PoolSize,
		prewarm:   config.Prewarm,
		cowClone:  config.CoWClone,
		instances: make([]SandboxInstance, 0, config.PoolSize),
		available: make(chan SandboxInstance, config.PoolSize),
	}
}

// Manage runs the pool management loop
func (p *SandboxPool) Manage(ctx context.Context) {
	log.Printf("Starting pool manager (size=%d, prewarm=%v)", p.size, p.prewarm)

	if p.prewarm {
		p.prewarmPool()
	}

	// Management loop
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.replenish()
		}
	}
}

// prewarmPool pre-warms the VM pool
func (p *SandboxPool) prewarmPool() {
	log.Printf("Pre-warming %d VMs...", p.size)

	// In real implementation, this would:
	// 1. Create MicroVM template
	// 2. Clone VMs using CoW
	// 3. Start VMs in background
	// 4. Add to available pool

	for i := 0; i < p.size; i++ {
		instance := SandboxInstance{
			ID:        fmt.Sprintf("vm-%d", i),
			Status:    "ready",
			VCPUs:     2,
			MemoryMB:  512,
			CreatedAt: time.Now(),
		}
		p.mu.Lock()
		p.instances = append(p.instances, instance)
		p.mu.Unlock()

		select {
		case p.available <- instance:
		default:
			log.Printf("Pool channel full, stopping prewarm at %d", i)
			return
		}
	}

	log.Printf("Pre-warm complete: %d VMs ready", len(p.instances))
}

// replenish ensures pool has enough available instances
func (p *SandboxPool) replenish() {
	p.mu.RLock()
	current := len(p.instances)
	p.mu.RUnlock()

	if current < p.size {
		needed := p.size - current
		log.Printf("Replenishing pool: need %d more VMs", needed)

		for i := 0; i < needed; i++ {
			instance := SandboxInstance{
				ID:        fmt.Sprintf("vm-%d", time.Now().UnixNano()),
				Status:    "ready",
				VCPUs:     2,
				MemoryMB:  512,
				CreatedAt: time.Now(),
			}
			p.mu.Lock()
			p.instances = append(p.instances, instance)
			p.mu.Unlock()

			select {
			case p.available <- instance:
			default:
				return
			}
		}
	}
}

// Acquire gets an instance from the pool
func (p *SandboxPool) Acquire() (*SandboxInstance, error) {
	select {
	case instance := <-p.available:
		instance.LastUsed = time.Now()
		p.mu.Lock()
		for i, inst := range p.instances {
			if inst.ID == instance.ID {
				p.instances[i].Status = "in-use"
				break
			}
		}
		p.mu.Unlock()
		return &instance, nil
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("pool exhausted: no available instances")
	}
}

// Release returns an instance to the pool
func (p *SandboxPool) Release(instance *SandboxInstance) {
	p.mu.Lock()
	for i, inst := range p.instances {
		if inst.ID == instance.ID {
			p.instances[i].Status = "ready"
			break
		}
	}
	p.mu.Unlock()

	p.available <- *instance
}

// Shutdown cleans up the pool
func (p *SandboxPool) Shutdown() {
	log.Printf("Shutting down pool (%d instances)", len(p.instances))
	p.mu.Lock()
	p.instances = p.instances[:0]
	close(p.available)
	p.mu.Unlock()
}

// GetStats returns pool statistics
func (p *SandboxPool) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ready := 0
	inUse := 0
	for _, inst := range p.instances {
		switch inst.Status {
		case "ready":
			ready++
		case "in-use":
			inUse++
		}
	}

	return map[string]interface{}{
		"total":      len(p.instances),
		"ready":      ready,
		"in_use":     inUse,
		"pool_size":  p.size,
		"prewarm":    p.prewarm,
		"cow_clone":  p.cowClone,
	}
}

func startAPIServer(addr string, pool *SandboxPool) {
	mux := http.NewServeMux()

	// Health checks
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ready"}`)
	})

	// Pool stats
	mux.HandleFunc("/api/v1/pool/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats := pool.GetStats()
		fmt.Fprintf(w, `{
			"total": %d,
			"ready": %d,
			"in_use": %d,
			"pool_size": %d,
			"prewarm": %v,
			"cow_clone": %v
		}`, stats["total"], stats["ready"], stats["in_use"], stats["pool_size"], stats["prewarm"], stats["cow_clone"])
	})

	// Acquire instance
	mux.HandleFunc("/api/v1/pool/acquire", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		instance, err := pool.Acquire()
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"id": "%s",
			"status": "%s",
			"vcpus": %d,
			"memory_mb": %d,
			"created_at": "%s",
			"last_used": "%s"
		}`, instance.ID, instance.Status, instance.VCPUs, instance.MemoryMB, instance.CreatedAt.Format(time.RFC3339), instance.LastUsed.Format(time.RFC3339))
	})

	// Release instance
	mux.HandleFunc("/api/v1/pool/release", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// In real implementation, read instance ID from request body
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"released"}`)
	})

	log.Printf("API server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}

func startMetricsServer(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	log.Printf("Metrics server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Metrics server failed: %v", err)
	}
}
