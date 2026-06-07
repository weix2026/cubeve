package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var Version = "dev"

func main() {
	var (
		listenAddr = flag.String("listen", ":8080", "REST API listen address")
		grpcAddr   = flag.String("grpc", ":50051", "gRPC listen address")
		metricsAddr = flag.String("metrics", ":9090", "Metrics listen address")
		incusEndpoint = flag.String("incus", "unix:///var/lib/incus/unix.socket", "Incus endpoint")
		// k8sConfig = flag.String("k8s-config", "", "Kubernetes config path")
	)
	flag.Parse()

	log.Printf("CubeAPI Gateway %s starting...", Version)
	log.Printf("  REST API: %s", *listenAddr)
	log.Printf("  gRPC: %s", *grpcAddr)
	log.Printf("  Metrics: %s", *metricsAddr)
	log.Printf("  Incus: %s", *incusEndpoint)

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

// Placeholder handlers
func listInstances(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"instances": []gin.H{},
		"count":     0,
	})
}

func createInstance(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{
		"id":     "inst-" + fmt.Sprintf("%d", time.Now().Unix()),
		"status": "creating",
	})
}

func getInstance(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "running",
		"runtime": "incus",
	})
}

func updateInstance(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "updated",
	})
}

func deleteInstance(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "deleted",
	})
}

func startInstance(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "starting",
	})
}

func stopInstance(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "stopping",
	})
}

func restartInstance(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "restarting",
	})
}

func createSnapshot(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusCreated, gin.H{
		"id":       id,
		"snapshot": "snap-" + fmt.Sprintf("%d", time.Now().Unix()),
		"status":   "created",
	})
}

func listSnapshots(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"instance":  id,
		"snapshots": []gin.H{},
	})
}

func listStoragePools(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"pools": []gin.H{},
	})
}

func createStoragePool(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{
		"name":   "pool-" + fmt.Sprintf("%d", time.Now().Unix()),
		"status": "created",
	})
}

func getStoragePool(c *gin.Context) {
	name := c.Param("name")
	c.JSON(http.StatusOK, gin.H{
		"name":   name,
		"driver": "zfs",
	})
}

func listNetworks(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"networks": []gin.H{},
	})
}

func createNetwork(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{
		"name":   "net-" + fmt.Sprintf("%d", time.Now().Unix()),
		"status": "created",
	})
}

func getNetwork(c *gin.Context) {
	name := c.Param("name")
	c.JSON(http.StatusOK, gin.H{
		"name":   name,
		"type":   "bridge",
	})
}

func listProfiles(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"profiles": []gin.H{},
	})
}

func createProfile(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{
		"name":   "profile-" + fmt.Sprintf("%d", time.Now().Unix()),
		"status": "created",
	})
}

func getProfile(c *gin.Context) {
	name := c.Param("name")
	c.JSON(http.StatusOK, gin.H{
		"name": name,
	})
}
