package scheduler

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Service struct {
	logger  *zap.Logger
	pgxPool *pgxpool.Pool
}

func NewService(logger *zap.Logger, pgxPool *pgxpool.Pool) *Service {
	return &Service{
		logger:  logger,
		pgxPool: pgxPool,
	}
}

var _ ServerInterface = (*Service)(nil)
