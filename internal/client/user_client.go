package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "article-service/proto"
)

// UserClient wraps gRPC connection to User Service
type UserClient struct {
	conn   *grpc.ClientConn
	client pb.UserServiceClient
}

// NewUserClient create new client
func NewUserClient(address string) (*UserClient, error) {
	// 1. Dial with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 2. Create connection
	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %w", err)
	}

	// 3. Create client stub
	client := pb.NewUserServiceClient(conn)

	return &UserClient{
		conn:   conn,
		client: client,
	}, nil
}

// Close connection
func (c *UserClient) Close() error {
	return c.conn.Close()
}

// GetUser call Service 1 to get user By ID
func (c *UserClient) GetUser(ctx context.Context, userId int32) (*pb.User, error) {
	// Set timeout for RPC call
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Call Service 1
	resp, err := c.client.GetUser(ctx, &pb.GetUserRequest{
		Id: userId,
	})
	if err != nil {
		return nil, fmt.Errorf("get user from service-1 failed: %w", err)
	}
	return resp, nil
}
