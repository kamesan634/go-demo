package database

import (
	"context"
	"fmt"
	"time"

	"github.com/go-demo/chat/internal/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func NewPostgres(cfg *config.DatabaseConfig, logger *zap.Logger) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg.GetDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	logger.Info("Connected to PostgreSQL",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.DBName),
	)

	return db, nil
}

// Close closes the database connection
func Close(db *sqlx.DB, logger *zap.Logger) {
	if err := db.Close(); err != nil {
		logger.Error("Error closing database connection", zap.Error(err))
	} else {
		logger.Info("Database connection closed")
	}
}
