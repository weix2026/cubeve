package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var Version = "dev"

func main() {
	var (
		listenAddr = flag.String("listen", ":8081", "Controller API listen address")
		apiGateway = flag.String("api-gateway", "http://localhost:8080", "API Gateway endpoint")
	)
	flag.Parse()

	log.Printf("Instance Controller %s starting...", Version)
	log.Printf("  API: %s", *listenAddr)
	log.Printf("  API Gateway: %s", *apiGateway)

	// Controller logic stub
	// TODO: Replace with actual K8s controller logic when dependencies are available
	go runControllerStub(*apiGateway)

	// HTTP API for health checks
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})
	mux.HandleFunc("/v1/instances", handleInstances)
	mux.HandleFunc("/v1/nodes", handleNodes)

	srv := &http.Server{
		Addr:    *listenAddr,
		Handler: mux,
	}

	go func() {
		log.Printf("Controller API listening on %s", *listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func runControllerStub(apiGateway string) {
	log.Println("Controller loop running (stub mode)")
	log.Println("Note: This is a stub implementation. K8s dependencies are not available.")
	log.Println("      Rebuild with full K8s dependencies for production use.")

	// Stub: periodically check API Gateway connection
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		resp, err := http.Get(apiGateway + "/healthz")
		if err != nil {
			log.Printf("API Gateway connection check failed: %v", err)
			continue
		}
		resp.Body.Close()
		log.Printf("API Gateway connection check: %s", resp.Status)
	}
}

func handleInstances(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		json.NewEncoder(w).Encode(map[string]interface{}{
			"instances": []string{},
			"count":     0,
			"note":      "Stub mode - K8s controller not active",
		})
	case "POST":
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "accepted",
			"note":   "Stub mode - no actual instance created",
		})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": []map[string]string{
			{"name": "node-1", "status": "ready"},
		},
		"count": 1,
		"note":  "Stub mode",
	})
}
