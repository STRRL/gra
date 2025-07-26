package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	gradv1 "github.com/strrl/gra/gen/grad/v1"
	grpcserver "github.com/strrl/gra/internal/grad/grpc"
	"github.com/strrl/gra/internal/grad/service"
)

var (
	httpPort string
	grpcPort string

	// Prometheus metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Duration of HTTP requests in seconds",
		},
		[]string{"method", "endpoint"},
	)

	grpcRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"method", "status"},
	)

	grpcRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "grpc_request_duration_seconds",
			Help: "Duration of gRPC requests in seconds",
		},
		[]string{"method"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(grpcRequestsTotal)
	prometheus.MustRegister(grpcRequestDuration)
}

var rootCmd = &cobra.Command{
	Use:   "grad",
	Short: "Grad - HTTP and gRPC service for managing runners",
	Long:  `Grad is a dual HTTP/gRPC service that manages runner lifecycle in Kubernetes.`,
	Run: func(cmd *cobra.Command, args []string) {
		runServers()
	},
}

func init() {
	rootCmd.Flags().StringVar(&httpPort, "http-port", "8080", "HTTP server port")
	rootCmd.Flags().StringVar(&grpcPort, "grpc-port", "9090", "gRPC server port")
}

func runServers() {
	var wg sync.WaitGroup
	wg.Add(2)

	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	config := service.LoadConfig()

	// Log current runner image configuration
	slog.Info("Starting grad service",
		"runner_image", config.Kubernetes.RunnerImage,
		"http_port", httpPort,
		"grpc_port", grpcPort,
	)

	// Initialize Kubernetes client
	k8sClient, err := service.NewKubernetesClient(config.Kubernetes)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Initialize runner service
	runnerService := service.NewRunnerService(k8sClient)

	// Create gRPC server with service dependency
	grpcSrv := grpcserver.NewServer(runnerService)

	// Start HTTP server
	go func() {
		defer wg.Done()
		runHTTPServer()
	}()

	// Start gRPC server
	go func() {
		defer wg.Done()
		runGRPCServer(grpcSrv)
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("Shutting down grad services...")

	// Graceful shutdown context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown both servers (we'll add this logic)
	shutdownServers(ctx)

	slog.Info("grad services stopped")
}

func runHTTPServer() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Add middleware for logging and recovery
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Add Prometheus metrics middleware
	r.Use(prometheusMiddleware())

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Readiness check endpoint
	r.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	server := &http.Server{
		Addr:    ":" + httpPort,
		Handler: r,
	}

	slog.Info("HTTP server starting", "port", httpPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("HTTP server error", "error", err)
	}
}

func runGRPCServer(srv *grpcserver.Server) {
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	gradv1.RegisterRunnerServiceServer(grpcServer, srv)

	// Enable reflection for grpcurl and other tools
	reflection.Register(grpcServer)

	slog.Info("gRPC server starting", "port", grpcPort)
	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("gRPC server error", "error", err)
	}
}

func shutdownServers(ctx context.Context) {
	// For now, we'll implement basic shutdown
	// In a production environment, you'd want to properly handle
	// graceful shutdown of both HTTP and gRPC servers
	slog.Info("Server shutdown logic would be implemented here")
}

func prometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := fmt.Sprintf("%d", c.Writer.Status())

		httpRequestsTotal.WithLabelValues(c.Request.Method, c.Request.URL.Path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, c.Request.URL.Path).Observe(duration)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
