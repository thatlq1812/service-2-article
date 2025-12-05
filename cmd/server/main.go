package main

import (
	"log"
	"net"

	"github.com/joho/godotenv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/thatlq1812/agrios-shared/pkg/common"
	"github.com/thatlq1812/service-2-article/internal/client"
	"github.com/thatlq1812/service-2-article/internal/config"
	"github.com/thatlq1812/service-2-article/internal/db"
	"github.com/thatlq1812/service-2-article/internal/repository"
	"github.com/thatlq1812/service-2-article/internal/server"
	pb "github.com/thatlq1812/service-2-article/proto"
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

	// 4. Setup Redis connection (for token blacklist check)
	redisClient, err := db.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Printf("Connected to Redis at %s", cfg.Redis.Addr)

	// 5. Create gRPC client to User Service (inter-service communication)
	userClient, err := client.NewUserClient(cfg.UserServiceAddr)
	if err != nil {
		log.Fatalf("Failed to connect to user service: %v", err)
	}
	log.Printf("Connected to User Service at %s", cfg.UserServiceAddr)

	// 6. Setup gRPC server
	grpcServer := grpc.NewServer()
	articleServer := server.NewArticleServer(articleRepo, userClient, redisClient, cfg.JWTSecret)
	pb.RegisterArticleServiceServer(grpcServer, articleServer)

	// 7. Enable reflection for tools like grpcurl
	reflection.Register(grpcServer)

	// 8. Setup TCP listener
	listener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	log.Printf("Article Service (gRPC) listening on port %s", cfg.GRPCPort)

	// 9. Start server in goroutine to handle graceful shutdown
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// 10. Wait for shutdown signal and perform graceful shutdown
	shutdownTimeout := common.GetEnvDuration("SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout)
	ctx := common.WaitForShutdown(shutdownTimeout)

	log.Println("Shutting down gRPC server...")
	grpcServer.GracefulStop()

	<-ctx.Done()
	log.Println("Server stopped gracefully")
}
