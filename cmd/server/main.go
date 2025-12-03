package main

import (
	"log"
	"net"

	"github.com/joho/godotenv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"agrios/pkg/common"
	"article-service/internal/client"
	"article-service/internal/config"
	"article-service/internal/db"
	"article-service/internal/repository"
	"article-service/internal/server"
	pb "article-service/proto"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}
	// 0. Load
	cfg := config.Load()

	// 1.

	// 2. Setup database connection pool
	pool, err := db.NewPostgresPool(cfg.DB)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to PostgreSQL successfully")

	// 3. Create repository
	articleRepo := repository.NewArticlePostgresRepository(pool)

	// 4. Create gRPC client to User Service (inter-service communication)
	userClient, err := client.NewUserClient(cfg.UserServiceAddr)
	if err != nil {
		log.Fatalf("Failed to connect to user service: %v", err)
	}
	log.Printf("Connected to User Service at %s", cfg.UserServiceAddr)

	// 5. Setup gRPC server
	grpcServer := grpc.NewServer()
	articleServer := server.NewArticleServiceServer(articleRepo, userClient)
	pb.RegisterArticleServiceServer(grpcServer, articleServer)

	// 6. Enable reflection for tools like grpcurl
	reflection.Register(grpcServer)

	// 7. Setup TCP listener
	listener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	log.Printf("Article Service (gRPC) listening on port %s", cfg.GRPCPort)

	// 8. Start server in goroutine to handle graceful shutdown
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// 9. Wait for shutdown signal and perform graceful shutdown
	shutdownTimeout := common.GetEnvDuration("SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout)
	ctx := common.WaitForShutdown(shutdownTimeout)

	log.Println("Shutting down gRPC server...")
	grpcServer.GracefulStop()

	<-ctx.Done()
	log.Println("Server stopped gracefully")
}
