package main

import (
	"log"
	"net"
	"time"

	"github.com/joho/godotenv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"agrios/pkg/common"
	"article-service/internal/client"
	"article-service/internal/db"
	"article-service/internal/repository"
	"article-service/internal/server"
	pb "article-service/proto"
)

const (
	defaultGRPCPort        = "50052"
	defaultUserServiceAddr = "localhost:50051"
	defaultShutdownTimeout = 10 * time.Second
)

// loadDBConfig loads database configuration from environment variables
func loadDBConfig() db.Config {
	return db.Config{
		Host:     common.GetEnvString("DB_HOST", "localhost"),
		Port:     common.GetEnvString("DB_PORT", "5432"),
		User:     common.MustGetEnvString("DB_USER"),
		Password: common.MustGetEnvString("DB_PASSWORD"),
		DBName:   common.MustGetEnvString("DB_NAME"),

		MaxConns:        common.GetEnvInt32("DB_MAX_CONNS", 10),
		MinConns:        common.GetEnvInt32("DB_MIN_CONNS", 2),
		MaxConnLifetime: common.GetEnvDuration("DB_MAX_CONN_LIFETIME", time.Hour),
		MaxConnIdleTime: common.GetEnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		ConnectTimeout:  common.GetEnvDuration("DB_CONNECT_TIMEOUT", 5*time.Second),
	}
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// 1. Load database configuration
	dbConfig := loadDBConfig()

	// 2. Setup database connection pool
	pool, err := db.NewPostgresPool(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to PostgreSQL successfully")

	// 3. Create repository
	articleRepo := repository.NewArticlePostgresRepository(pool)

	// 4. Create gRPC client to User Service (inter-service communication)
	userServiceAddr := common.GetEnvString("USER_SERVICE_ADDR", defaultUserServiceAddr)
	userClient, err := client.NewUserClient(userServiceAddr)
	if err != nil {
		log.Fatalf("Failed to connect to user service: %v", err)
	}
	log.Printf("Connected to User Service at %s", userServiceAddr)

	// 5. Setup gRPC server
	grpcServer := grpc.NewServer()
	articleServer := server.NewArticleServiceServer(articleRepo, userClient)
	pb.RegisterArticleServiceServer(grpcServer, articleServer)

	// 6. Enable reflection for tools like grpcurl
	reflection.Register(grpcServer)

	// 7. Setup TCP listener
	grpcPort := common.GetEnvString("GRPC_PORT", defaultGRPCPort)
	listener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	log.Printf("Article Service (gRPC) listening on port %s", grpcPort)

	// 8. Start server in goroutine to handle graceful shutdown
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// 9. Wait for shutdown signal and perform graceful shutdown
	shutdownTimeout := common.GetEnvDuration("SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	ctx := common.WaitForShutdown(shutdownTimeout)

	log.Println("Shutting down gRPC server...")
	grpcServer.GracefulStop()

	<-ctx.Done()
	log.Println("Server stopped gracefully")
}
