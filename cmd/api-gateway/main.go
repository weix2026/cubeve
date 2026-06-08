package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var Version = "dev"

var incusClient *IncusClient

func main() {
	var (
		listenAddr    = flag.String("listen", ":8080", "REST API listen address")
		grpcAddr      = flag.String("grpc", ":50051", "gRPC listen address")
		metricsAddr   = flag.String("metrics", ":9090", "Metrics listen address")
		incusEndpoint = flag.String("incus", "unix:///var/lib/incus/unix.socket", "Incus endpoint")
	)
	flag.Parse()

	log.Printf("CubeAPI Gateway %s starting...", Version)
	log.Printf("  REST API: %s", *listenAddr)
	log.Printf("  gRPC: %s", *grpcAddr)
	log.Printf("  Metrics: %s", *metricsAddr)
	log.Printf("  Incus: %s", *incusEndpoint)

	// Initialize Incus client
	incusClient = NewIncusClient(*incusEndpoint, "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := incusClient.GetServerInfo(ctx); err != nil {
		log.Printf("Warning: Incus connection failed: %v", err)
	} else {
		log.Printf("Incus connection established")
	}

	// REST API server
	go startRESTServer(*listenAddr)

	// Metrics server
	go startMetricsServer(*metricsAddr)

	// Wait for shutdown signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down...")
	os.Exit(0)
}

func startRESTServer(addr string) {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Health checks
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, err := incusClient.GetServerInfo(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		v1.GET("/instances", listInstances)
		v1.POST("/instances", createInstance)
		v1.GET("/instances/:id", getInstance)
		v1.PATCH("/instances/:id", updateInstance)
		v1.DELETE("/instances/:id", deleteInstance)
		v1.POST("/instances/:id/start", startInstance)
		v1.POST("/instances/:id/stop", stopInstance)
		v1.POST("/instances/:id/restart", restartInstance)
		v1.POST("/instances/:id/snapshot", createSnapshot)
		v1.GET("/instances/:id/snapshots", listSnapshots)
	}

	// Storage routes
	v1.GET("/storage-pools", listStoragePools)
	v1.POST("/storage-pools", createStoragePool)
	v1.GET("/storage-pools/:name", getStoragePool)

	// Network routes
	v1.GET("/networks", listNetworks)
	v1.POST("/networks", createNetwork)
	v1.GET("/networks/:name", getNetwork)

	// Profile routes
	v1.GET("/profiles", listProfiles)
	v1.POST("/profiles", createProfile)
	v1.GET("/profiles/:name", getProfile)

	log.Printf("REST API server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("REST API server failed: %v", err)
	}
}

func startMetricsServer(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	log.Printf("Metrics server listening on %s", addr)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Metrics server failed: %v", err)
	}
}

// Instance handlers
func listInstances(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	names, err := incusClient.ListInstances(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	instances := make([]gin.H, 0, len(names))
	for _, fullPath := range names {
		// Incus returns paths like "/1.0/instances/name", extract just the name
		name := fullPath
		if idx := strings.LastIndex(fullPath, "/"); idx != -1 {
			name = fullPath[idx+1:]
		}
		inst, err := incusClient.GetInstance(ctx, name)
		if err != nil {
			continue
		}
		instances = append(instances, gin.H{
			"name":         inst.Name,
			"status":       inst.Status,
			"type":         inst.Type,
			"architecture": inst.Architecture,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"instances": instances,
		"count":     len(instances),
	})
}

func createInstance(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := incusClient.CreateInstance(ctx, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":     req["name"],
		"status": "created",
	})
}

func getInstance(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inst, err := incusClient.GetInstance(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":         inst.Name,
		"status":       inst.Status,
		"type":         inst.Type,
		"architecture": inst.Architecture,
		"config":       inst.Config,
		"devices":      inst.Devices,
	})
}

func updateInstance(c *gin.Context) {
	id := c.Param("id")
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := incusClient.CreateInstance(ctx, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "updated",
	})
}

func deleteInstance(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := incusClient.DeleteInstance(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "deleted",
	})
}

func startInstance(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := incusClient.StartInstance(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "starting",
	})
}

func stopInstance(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := incusClient.StopInstance(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "stopping",
	})
}

func restartInstance(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := incusClient.RestartInstance(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "restarting",
	})
}

func createSnapshot(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Name = "snap-" + fmt.Sprintf("%d", time.Now().Unix())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := incusClient.CreateSnapshot(ctx, id, req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       id,
		"snapshot": req.Name,
		"status":   "created",
	})
}

func listSnapshots(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	snaps, err := incusClient.ListSnapshots(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instance":  id,
		"snapshots": snaps,
		"count":     len(snaps),
	})
}

// Storage handlers
func listStoragePools(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pools, err := incusClient.ListStoragePools(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pools": pools,
		"count": len(pools),
	})
}

func createStoragePool(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := incusClient.CreateStoragePool(ctx, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"name":   req["name"],
		"status": "created",
	})
}

func getStoragePool(c *gin.Context) {
	name := c.Param("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := incusClient.GetStoragePool(ctx, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":   pool.Name,
		"driver": pool.Driver,
		"config": pool.Config,
	})
}

// Network handlers
func listNetworks(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nets, err := incusClient.ListNetworks(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"networks": nets,
		"count":    len(nets),
	})
}

func createNetwork(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := incusClient.CreateNetwork(ctx, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"name":   req["name"],
		"status": "created",
	})
}

func getNetwork(c *gin.Context) {
	name := c.Param("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	net, err := incusClient.GetNetwork(ctx, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":   net.Name,
		"type":   net.Type,
		"config": net.Config,
	})
}

// Profile handlers
func listProfiles(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	profiles, err := incusClient.ListProfiles(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"profiles": profiles,
		"count":    len(profiles),
	})
}

func createProfile(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := incusClient.CreateProfile(ctx, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"name":   req["name"],
		"status": "created",
	})
}

func getProfile(c *gin.Context) {
	name := c.Param("name")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	profile, err := incusClient.GetProfile(ctx, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":    profile.Name,
		"config":  profile.Config,
		"devices": profile.Devices,
	})
}
