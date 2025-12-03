package config

import (
	"agrios/pkg/common"
	"service-2-article/internal/db"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	GRPCPort        string
	ShutdownTimeout time.Duration
	UserServiceAddr string

	DB        db.Config
	JWTSecret string
}

func Load() *Config {
	return &Config{
		// Server Config
		GRPCPort:        common.GetEnvString("GRPC_PORT", "50052"),
		ShutdownTimeout: common.GetEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		UserServiceAddr: common.GetEnvString("USER_SERVICE_ADDR", "localhost:50051"),

		// JWT
		JWTSecret: common.GetEnvString("JWT_SECRET", "insecure-default-secret-change-this"), // default value for Dev

		// Database Config
		DB: db.Config{
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
		},
	}
}
