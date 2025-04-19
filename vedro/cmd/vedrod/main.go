package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"vedro/config"
	"vedro/internal/server"
	"vedro/internal/storage"
)

var (
	version    = "dev"
	showVersion bool
	showConfig  bool
)

func init() {
	flag.BoolVar(&showVersion, "v", false, "Show version info")
	flag.BoolVar(&showVersion, "version", false, "Show version info")
	flag.BoolVar(&showConfig, "c", false, "Show current configuration")
	flag.BoolVar(&showConfig, "config", false, "Show current configuration")

	flag.Usage = customUsage
}

func customUsage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "  -c -config\tShow current configuration\n")
	fmt.Fprintf(flag.CommandLine.Output(), "  -v -version\tShow version info\n")
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Printf("vedrod version %s — (c) Qwaderton, 2025\n", version)
		os.Exit(0)
	}

	if showConfig {
		printConfig()
		os.Exit(0)
	}

	cache := storage.NewCache()
	cache.Scan(config.RootPath)

	ticker := time.NewTicker(config.ScanInterval * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			log.Println("Starting periodic scan")
			cache.Scan(config.RootPath)
			log.Println("Periodic scan completed")
		}
	}()

	handler := server.NewHandler(cache)
	server := &http.Server{
		Addr:    config.ServerAddr,
		Handler: handler,
	}

	go func() {
		log.Println("Server starting " + config.ServerAddr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
	log.Println("Server stopped")
}


func printConfig() {
	fmt.Println("Current Configuration:")
	fmt.Printf("Root Path:      %s\n", config.RootPath)
	fmt.Printf("Server Addr:    %s\n", config.ServerAddr)
	fmt.Printf("Scan Interval:  %d seconds\n", config.ScanInterval)
	fmt.Printf("Enable Recover: %t\n", config.EnableRecover)
	fmt.Printf("Enable Logging: %t\n", config.EnableLogging)
}