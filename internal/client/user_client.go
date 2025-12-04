package client

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	userpb "service-1-user/proto"
	"service-2-article/internal/response"
)

const (
	defaultTimeout      = 3 * time.Second
	connectionTimeout   = 5 * time.Second
	maxRetries          = 3
	retryBackoffInitial = 100 * time.Millisecond
	retryBackoffMax     = 1 * time.Second
)

type UserClient struct {
	client userpb.UserServiceClient
	conn   *grpc.ClientConn
}

// NewUserClient creates a new gRPC client connection to User Service
// Blocks until connection is established or timeout occurs
func NewUserClient(address string) (*UserClient, error) {
	log.Printf("[UserClient] Connecting to user service at %s", address)

	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("[UserClient] Failed to connect to user service: address=%s, error=%v", address, err)
		return nil, response.GRPCError(codes.Unavailable, "Failed to connect to user service. Check network and service availability.")
	}

	log.Printf("[UserClient] Successfully connected to user service at %s", address)
	return &UserClient{
		client: userpb.NewUserServiceClient(conn),
		conn:   conn,
	}, nil
}

// GetUser calls User Service to retrieve a user by ID with proper error handling
// Accepts int32 for compatibility with Article Service proto definitions
func (c *UserClient) GetUser(ctx context.Context, userID int32) (*userpb.User, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	log.Printf("[UserClient.GetUser] Calling user service: user_id=%d", userID)

	resp, err := c.client.GetUser(ctx, &userpb.GetUserRequest{Id: userID})
	if err != nil {
		return c.handleGetUserError(err, userID)
	}

	// Extract user from wrapped response
	if resp.GetData() == nil || resp.GetData().GetUser() == nil {
		log.Printf("[UserClient.GetUser] User not found: user_id=%d", userID)
		return nil, response.GRPCError(codes.NotFound, "User not found. Verify the user ID exists.")
	}

	user := resp.GetData().GetUser()
	log.Printf("[UserClient.GetUser] Success: user_id=%d, email=%s", user.Id, user.Email)
	return user, nil
}

// handleGetUserError processes errors from GetUser call
func (c *UserClient) handleGetUserError(err error, userID int32) (*userpb.User, error) {
	st, ok := status.FromError(err)
	if !ok {
		log.Printf("[UserClient.GetUser] Unknown error: user_id=%d, error=%v", userID, err)
		return nil, response.GRPCError(codes.Internal, "Unknown error from user service. Contact support if the issue persists.")
	}

	switch st.Code() {
	case codes.NotFound:
		log.Printf("[UserClient.GetUser] User not found: user_id=%d", userID)
		return nil, response.GRPCError(codes.NotFound, "User not found. Verify the user ID exists.")

	case codes.InvalidArgument:
		log.Printf("[UserClient.GetUser] Invalid argument: user_id=%d, error=%s", userID, st.Message())
		return nil, response.GRPCError(codes.InvalidArgument, "Invalid user ID. Provide a valid ID greater than 0.")

	case codes.DeadlineExceeded:
		log.Printf("[UserClient.GetUser] Timeout: user_id=%d, timeout=%v", userID, defaultTimeout)
		return nil, response.GRPCError(codes.DeadlineExceeded, "User service timeout. Please try again.")

	case codes.Unavailable:
		log.Printf("[UserClient.GetUser] Service unavailable: user_id=%d", userID)
		return nil, response.GRPCError(codes.Unavailable, "User service is currently unavailable. Check service status.")

	case codes.Internal:
		log.Printf("[UserClient.GetUser] Internal error: user_id=%d, error=%s", userID, st.Message())
		return nil, response.GRPCError(codes.Internal, "User service internal error. Contact support if the issue persists.")

	default:
		log.Printf("[UserClient.GetUser] Unexpected error: user_id=%d, code=%s, message=%s", userID, st.Code(), st.Message())
		return nil, response.GRPCError(codes.Unknown, "Unexpected error from user service. Contact support if the issue persists.")
	}
}

// GetUserWithRetry retrieves a user with retry logic for transient failures
func (c *UserClient) GetUserWithRetry(ctx context.Context, userID int32) (*userpb.User, error) {
	var lastErr error
	backoff := retryBackoffInitial

	for attempt := 1; attempt <= maxRetries; attempt++ {
		user, err := c.GetUser(ctx, userID)
		if err == nil {
			return user, nil
		}

		lastErr = err
		st := status.Convert(err)

		if !isRetryableError(st.Code()) {
			log.Printf("[UserClient.GetUserWithRetry] Non-retryable error: user_id=%d, code=%s, attempt=%d", userID, st.Code(), attempt)
			return nil, err
		}

		if attempt < maxRetries {
			log.Printf("[UserClient.GetUserWithRetry] Retrying after error: user_id=%d, attempt=%d/%d, backoff=%v", userID, attempt, maxRetries, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > retryBackoffMax {
				backoff = retryBackoffMax
			}
		}
	}

	log.Printf("[UserClient.GetUserWithRetry] All retries exhausted: user_id=%d, attempts=%d", userID, maxRetries)
	return nil, lastErr
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(code codes.Code) bool {
	switch code {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

// ValidateToken validates a JWT token
func (c *UserClient) ValidateToken(ctx context.Context, token string) (*userpb.ValidateTokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	log.Printf("[UserClient.ValidateToken] Validating token")

	resp, err := c.client.ValidateToken(ctx, &userpb.ValidateTokenRequest{Token: token})
	if err != nil {
		st := status.Convert(err)
		log.Printf("[UserClient.ValidateToken] Error: code=%s, message=%s", st.Code(), st.Message())
		return nil, err
	}

	// Extract validation data from wrapped response
	if resp.GetData() == nil {
		log.Printf("[UserClient.ValidateToken] Invalid response structure")
		return nil, response.GRPCError(codes.Internal, "Invalid response from user service. Contact support if the issue persists.")
	}

	if resp.GetData().GetValid() {
		log.Printf("[UserClient.ValidateToken] Token valid: user_id=%d, email=%s", resp.GetData().GetUserId(), resp.GetData().GetEmail())
	} else {
		log.Printf("[UserClient.ValidateToken] Token invalid")
	}

	return resp, nil
}

// Close closes the gRPC connection to User Service
func (c *UserClient) Close() error {
	if c.conn != nil {
		log.Printf("[UserClient] Closing connection to user service")
		return c.conn.Close()
	}
	return nil
}

// HealthCheck performs a simple health check on the user service
func (c *UserClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := c.client.GetUser(ctx, &userpb.GetUserRequest{Id: -1})
	if err != nil {
		st := status.Convert(err)
		if st.Code() == codes.NotFound || st.Code() == codes.InvalidArgument {
			return nil
		}
		log.Printf("[UserClient.HealthCheck] Health check failed: code=%s", st.Code())
		return err
	}

	return nil
}
