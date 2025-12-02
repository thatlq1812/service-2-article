package main

import (
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"article-service/internal/client"
	"article-service/internal/db"
	"article-service/internal/repository"
	"article-service/internal/server"
	pb "article-service/proto"
)

func mustGetEnvInt32(key string, defaultValue int32) int32 {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.ParseInt(valStr, 10, 32)
	if err != nil {
		log.Printf("W: could not parse %s='%s' to int32. Using default value %d.", key, valStr, defaultValue)
		return defaultValue
	}
	return int32(val)
}

func mustGetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := time.ParseDuration(valStr)
	if err != nil {
		log.Fatalf("Error: Could not parse %s='%s' to time.Duration. Example format: 1h, 30m, 5s.", key, valStr)
		return defaultValue
	}
	return val
}

func LoadDBConfig() db.Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found.")
	}

	return db.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),

		MaxConns:        mustGetEnvInt32("DB_MAX_CONNS", 10),
		MinConns:        mustGetEnvInt32("DB_MIN_CONNS", 2),
		MaxConnLifetime: mustGetEnvDuration("DB_MAX_CONN_LIFETIME", time.Hour),
		MaxConnIdleTime: mustGetEnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		ConnectTimeout:  mustGetEnvDuration("DB_CONNECT_TIMEOUT", 5*time.Second),
	}
}

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found.")
	}
	// 1. Setup database connection
	dbConfig := db.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),

		MaxConns:        mustGetEnvInt32("DB_MAX_CONNS", 10),
		MinConns:        mustGetEnvInt32("DB_MIN_CONNS", 2),
		MaxConnLifetime: mustGetEnvDuration("DB_MAX_CONN_LIFETIME", time.Hour),
		MaxConnIdleTime: mustGetEnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		ConnectTimeout:  mustGetEnvDuration("DB_CONNECT_TIMEOUT", 5*time.Second),
	}

	pool, err := db.NewPostgresPool(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to PostgreSQL")

	// 2. Create repository
	articleRepo := repository.NewArticlePostgresRepository(pool)

	// 3. Create gRPC client to User Service (Service-1)
	userClient, err := client.NewUserClient("localhost:50051")
	if err != nil {
		log.Fatalf("Failed to connect to user service: %v", err)
	}
	defer userClient.Close()
	log.Println("Connected to User Service at localhost:50051")

	// 4. Create gRPC server
	grpcServer := grpc.NewServer()
	articleServer := server.NewArticleServiceServer(articleRepo, userClient)
	pb.RegisterArticleServiceServer(grpcServer, articleServer)

	// Enable reflection for grpcurl
	reflection.Register(grpcServer)

	// 5. Start listening
	listenPort := os.Getenv("GRCP_PORT")
	if listenPort == "" {
		listenPort = "50052"
	}

	listener, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Println("Article Service listening on :50052")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
