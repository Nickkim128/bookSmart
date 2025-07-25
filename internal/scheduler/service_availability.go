package scheduler

import (
	"context"
	_ "embed"
	"net/http"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	availabilityRequest := Availability{}
	if err := c.ShouldBindJSON(&availabilityRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var (
		orgID   = "00000000-0000-0000-0000-000000000001"
		role    = UserRoleStudent
		matched = false
		now     = time.Now()
	)

	chunks, err := convertIntervalsIntoChunks(availabilityRequest.AvailableTimeIntervals)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(chunks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid time intervals provided"})
		return
	}

	err = batchUpsertAvailability(c.Request.Context(), s.pgxPool, userID, orgID, role, matched, now, chunks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Availability created successfully"})
}

//go:embed queries/availibility/get_availibility.sql
var queryGetAvailabilitySQL string

//go:embed queries/availibility/batch_upsert_availability.sql
var queryBatchUpsertAvailabilitySQL string

func (s *Service) GetAvailability(c *gin.Context, userID string) {
	availabilityRecords, err := getAvailability(c.Request.Context(), s.pgxPool, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert records to TimeInterval chunks
	chunks := make([]TimeInterval, len(availabilityRecords))
	for i, availability := range availabilityRecords {
		chunks[i] = TimeInterval{
			availability.StartTime,
			availability.EndTime,
		}
	}

	// Group consecutive chunks back into larger intervals
	availabileTimeIntervals := groupConsecutiveChunks(chunks)

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
	request := BatchAvailabilityRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := []Availability{}
	for _, userID := range request.UserIds {
		availabilityRecords, err := getAvailability(c.Request.Context(), s.pgxPool, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Convert records to TimeInterval chunks
		chunks := make([]TimeInterval, len(availabilityRecords))
		for i, availability := range availabilityRecords {
			chunks[i] = TimeInterval{
				availability.StartTime,
				availability.EndTime,
			}
		}

		// Group consecutive chunks back into larger intervals
		availableTimeIntervals := groupConsecutiveChunks(chunks)

		response = append(response, Availability{
			AvailableTimeIntervals: availableTimeIntervals,
			UserId:                 userID,
		})
	}

	c.JSON(http.StatusOK, response)
}

func getAvailability(ctx context.Context, pgxPool *pgxpool.Pool, userID string) ([]AvailabilityRecord, error) {
	availability := []AvailabilityRecord{}
	return availability, pgxscan.Select(ctx, pgxPool, &availability, queryGetAvailabilitySQL, userID)
}

func batchUpsertAvailability(ctx context.Context, pgxPool *pgxpool.Pool, userID, orgID string, role UserRole, matched bool, now time.Time, chunks []TimeInterval) error {
	batch := &pgx.Batch{}

	for _, interval := range chunks {
		availabilityID := uuid.New().String()
		batch.Queue(queryBatchUpsertAvailabilitySQL,
			availabilityID,
			orgID,
			userID,
			role,
			interval[0],
			interval[1],
			matched,
			now,
			now,
		)
	}

	batchResult := pgxPool.SendBatch(ctx, batch)
	defer func() {
		_ = batchResult.Close()
	}()

	for i := 0; i < len(chunks); i++ {
		_, err := batchResult.Exec()
		if err != nil {
			return err
		}
	}

	return nil
}

// type additionalAvailabilityRecord struct {
// 	OrgID     string    `json:"org_id"`
// 	Role      string    `json:"role"`
// 	Matched   bool      `json:"matched"`
// 	CreatedAt time.Time `json:"created_at"`
// 	UpdatedAt time.Time `json:"updated_at"`
// }

// //go:embed queries/availibility/create_availibility.sql
// var queryCreateAvailabilitySQL string

// func createAvailability(ctx context.Context, pgxPool *pgxpool.Pool, availability AvailabilityRecord, additionalAvailability additionalAvailabilityRecord) error {
// 	_, err := pgxPool.Exec(ctx,
// 		queryCreateAvailabilitySQL,
// 		availability.AvailabilityID,
// 		additionalAvailability.OrgID,
// 		availability.UserID,
// 		additionalAvailability.Role,
// 		availability.StartTime,
// 		availability.EndTime,
// 		additionalAvailability.Matched,
// 		additionalAvailability.CreatedAt,
// 		additionalAvailability.UpdatedAt,
// 	)
// 	return err
// }
