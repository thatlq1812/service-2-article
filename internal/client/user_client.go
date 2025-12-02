package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "article-service/proto"
)

// UserClient wraps gRPC connection to User Service for inter-service communication
type UserClient struct {
	conn   *grpc.ClientConn
	client pb.UserServiceClient
}

// NewUserClient creates a new gRPC client connection to User Service
// Blocks until connection is established or timeout occurs
func NewUserClient(address string) (*UserClient, error) {
	// Create connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Establish gRPC connection
	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service at %s: %w", address, err)
	}

	// Create client stub
	client := pb.NewUserServiceClient(conn)

	return &UserClient{
		conn:   conn,
		client: client,
	}, nil
}

// Close closes the gRPC connection to User Service
func (c *UserClient) Close() error {
	return c.conn.Close()
}

// GetUser calls User Service to retrieve a user by ID
func (c *UserClient) GetUser(ctx context.Context, userId int32) (*pb.User, error) {
	// Set timeout for RPC call
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Call User Service - returns GetUserResponse per proto definition
	resp, err := c.client.GetUser(ctx, &pb.GetUserRequest{
		Id: userId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user from user service: %w", err)
	}

	return resp.User, nil
}
