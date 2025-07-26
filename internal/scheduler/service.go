package scheduler

import (
	"database/sql"
	"scheduler-api/internal/auth"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Service struct {
	logger          *zap.Logger
	pgxPool         *pgxpool.Pool
	sqlDB           *sql.DB
	firebaseService *auth.FirebaseService
}

func NewService(logger *zap.Logger, pgxPool *pgxpool.Pool, sqlDB *sql.DB, firebaseService *auth.FirebaseService) *Service {
	return &Service{
		logger:          logger,
		pgxPool:         pgxPool,
		sqlDB:           sqlDB,
		firebaseService: firebaseService,
	}
}

var _ ServerInterface = (*Service)(nil)
