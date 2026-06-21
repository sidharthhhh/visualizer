package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/containerscope/backend/internal/config"
	"github.com/containerscope/backend/internal/db"
	"github.com/containerscope/backend/internal/logger"
	"github.com/containerscope/backend/internal/server"
	"github.com/containerscope/backend/proto/agent"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New("info")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dsn := cfg.Database.DSN()

	if err := db.RunMigrations(dsn, log); err != nil {
		log.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		log.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	httpServer := server.New(log, pool, cfg)

	httpAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         httpAddr,
		Handler:      httpServer.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("HTTP server starting", "addr", httpAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	grpcAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port+1)
	grpcServer := grpc.NewServer()
	agentServer := agent.NewAgentServer(pool, log, httpServer.Hub())
	agent.RegisterAgentServiceServer(grpcServer, agentServer)

	go func() {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			log.Error("gRPC listen error", "error", err)
			os.Exit(1)
		}
		log.Info("gRPC server starting", "addr", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("gRPC server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gracefully")

	grpcServer.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server forced to shutdown", "error", err)
		os.Exit(1)
	}

	log.Info("server stopped")
}
