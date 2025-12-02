package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string

	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
}

func NewPostgresPool(cfg Config) (*pgxpool.Pool, error) {
	// Config string
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName,
	)

	// Config pool
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("Parse congif failed: %v", err)
	}

	// Define connection settings
	config.MaxConns = cfg.MaxConns
	config.MinConns = cfg.MinConns
	config.MaxConnLifetime = cfg.MaxConnLifetime
	config.MaxConnIdleTime = cfg.MaxConnIdleTime

	timeout := cfg.ConnectTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// Create pool with 5 seconds timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("Connect to database failed: %w", err)
	}

	// Pring to test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("Ping to database failed : %w", err)
	}

	return pool, nil
}
