package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"article-service/internal/client"
	"article-service/internal/db"
	"article-service/internal/repository"
	"article-service/internal/server"
	pb "article-service/proto"
)

func main() {
	// 1. Setup database connection
	dbConfig := db.Config{
		Host:     "127.0.0.1",
		Port:     "5432",
		User:     "postgres",
		Password: "postgres",
		DBName:   "agrios_articles",
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
	listener, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Println("Article Service listening on :50052")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
