package scheduler

import (
	"context"
	_ "embed"
	"net/http"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AvailabilityService interface {
	CreateAvailability(*gin.Context, string)
	GetAvailability(*gin.Context, string)
	UpdateAvailability(*gin.Context, string)
	GetBatchAvailability(*gin.Context)
}

var _ AvailabilityService = (*Service)(nil)

type AvailabilityRecord struct {
	AvailabilityID string    `json:"availability_id"`
	UserID         string    `json:"user_id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
}

func (s *Service) CreateAvailability(c *gin.Context, userID string) {
	// TODO: Implement availability creation logic
	// - Get user ID from context
	// - Validate request body
	// - Create availability record
	// - Return created availability
}

//go:embed queries/availibility/get_availibility.sql
var queryGetAvailabilitySQL string

func (s *Service) GetAvailability(c *gin.Context, userID string) {
	availabilityRecords, err := getAvailability(c.Request.Context(), s.pgxPool, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	availabileTimeIntervals := make([]TimeInterval, len(availabilityRecords))
	for i, availability := range availabilityRecords {
		availabileTimeIntervals[i] = TimeInterval{
			availability.StartTime,
			availability.EndTime,
		}
	}

	c.JSON(http.StatusOK, Availability{
		AvailableTimeIntervals: availabileTimeIntervals,
		UserId:                 userID,
	})
}

func (s *Service) UpdateAvailability(c *gin.Context, userID string) {
	// TODO: Implement availability update logic
	// - Get user ID from context
	// - Get existing availability
	// - Apply add/remove changes
	// - Validate updated time intervals
	// - Save to database
	// - Return updated availability
}

func (s *Service) GetBatchAvailability(c *gin.Context) {
	// TODO: Implement batch availability retrieval logic
	// - Query database for multiple users' availability
	// - Return array of availability records
}

func getAvailability(ctx context.Context, pgxPool *pgxpool.Pool, userID string) ([]AvailabilityRecord, error) {
	availability := []AvailabilityRecord{}
	return availability, pgxscan.Select(ctx, pgxPool, &availability, queryGetAvailabilitySQL, userID)
}
