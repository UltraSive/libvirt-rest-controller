package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"libvirt-controller/internal/metrics"
	"libvirt-controller/internal/server"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func gracefulShutdown(apiServer *http.Server, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

func main() {
	apiServer := server.NewServer()

	// Register your libvirt collector
	interfaceCollector := metrics.NewLibvirtInterfaceCollector()
	prometheus.MustRegister(interfaceCollector)
	diskCollector := metrics.NewLibvirtDiskCollector()
	prometheus.MustRegister(diskCollector)

	// Metrics server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:    ":9100",
		Handler: metricsMux,
	}

	// Graceful shutdown done channel
	done := make(chan bool, 1)

	// Run graceful shutdown for API and Metrics servers
	go gracefulShutdown(apiServer, done)
	go gracefulShutdown(metricsServer, done)

	// Start servers
	go func() {
		log.Println("API server listening on :8080")
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("API server error: %v", err)
		}
	}()

	go func() {
		log.Println("Metrics server listening on :9100")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Metrics server error: %v", err)
		}
	}()

	// Wait for shutdown
	<-done
	<-done
	log.Println("All servers shut down cleanly.")
}
