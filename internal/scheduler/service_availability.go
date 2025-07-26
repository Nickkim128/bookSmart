package scheduler

import (
	"context"
	"net/http"
	"scheduler-api/internal/auth"
	"time"

	_ "embed"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
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
	// Validate Firebase token and get user claims
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Users can only create availability for themselves (unless admin)
	if currentUser.UserID != userID && currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Can only create availability for yourself",
		})
		return
	}

	// Parse request body
	var req Availability
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate that the user_id in the request matches the URL parameter
	if req.UserId != userID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "user_id_mismatch",
			"message": "User ID in request body must match URL parameter",
		})
		return
	}

	// Validate time intervals
	if len(req.AvailableTimeIntervals) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "no_intervals",
			"message": "At least one time interval is required",
		})
		return
	}

	// Convert intervals into smaller chunks for database storage
	chunks, err := convertIntervalsIntoChunks(req.AvailableTimeIntervals)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_intervals",
			"message": "Invalid time intervals provided",
			"details": err.Error(),
		})
		return
	}

	if len(chunks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "no_valid_intervals",
			"message": "No valid time intervals after processing",
		})
		return
	}

	// Store availability in database
	err = batchUpsertAvailability(c.Request.Context(), s.pgxPool, userID, currentUser.OrgID, UserRole(currentUser.Role), false, time.Now(), chunks)
	if err != nil {
		s.logger.Error("Failed to create availability", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to create availability",
		})
		return
	}

	// Log successful creation
	s.logger.Info("Availability created successfully",
		zap.String("user_id", userID),
		zap.String("created_by", currentUser.UserID),
		zap.Int("interval_count", len(chunks)),
	)

	// Return created availability
	response := Availability{
		UserId:                 userID,
		AvailableTimeIntervals: req.AvailableTimeIntervals,
	}

	c.JSON(http.StatusCreated, response)
}

//go:embed queries/availibility/get_availibility.sql
var queryGetAvailabilitySQL string

//go:embed queries/availibility/batch_upsert_availability.sql
var queryBatchUpsertAvailabilitySQL string

func (s *Service) GetAvailability(c *gin.Context, userID string) {
	// Validate Firebase token and get user claims
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Users can only view their own availability (unless admin)
	if currentUser.UserID != userID && currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Can only view your own availability",
		})
		return
	}

	// Get availability records from database
	availabilityRecords, err := getAvailability(c.Request.Context(), s.pgxPool, userID)
	if err != nil {
		s.logger.Error("Failed to get availability", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to retrieve availability",
		})
		return
	}

	// Convert database records to TimeInterval chunks
	chunks := make([]TimeInterval, len(availabilityRecords))
	for i, availability := range availabilityRecords {
		chunks[i] = TimeInterval{
			availability.StartTime,
			availability.EndTime,
		}
	}

	// Group consecutive chunks back into larger intervals for API response
	availableTimeIntervals := groupConsecutiveChunks(chunks)

	response := Availability{
		UserId:                 userID,
		AvailableTimeIntervals: availableTimeIntervals,
	}

	c.JSON(http.StatusOK, response)
}

func (s *Service) UpdateAvailability(c *gin.Context, userID string) {
	// Validate Firebase token and get user claims
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Users can only update their own availability (unless admin)
	if currentUser.UserID != userID && currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Can only update your own availability",
		})
		return
	}

	// Parse request body
	var req AvailabilityUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate that the user_id in the request matches the URL parameter
	if req.UserId != userID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "user_id_mismatch",
			"message": "User ID in request body must match URL parameter",
		})
		return
	}

	// Get current availability
	currentAvailabilityRecords, err := getAvailability(c.Request.Context(), s.pgxPool, userID)
	if err != nil {
		s.logger.Error("Failed to get current availability", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to retrieve current availability",
		})
		return
	}

	// Convert current records to TimeInterval chunks
	currentChunks := make([]TimeInterval, len(currentAvailabilityRecords))
	for i, availability := range currentAvailabilityRecords {
		currentChunks[i] = TimeInterval{
			availability.StartTime,
			availability.EndTime,
		}
	}

	// Group current chunks into intervals
	currentIntervals := groupConsecutiveChunks(currentChunks)

	// Apply changes (add and remove)
	updatedIntervals := currentIntervals

	// Remove intervals
	if req.Changes.Remove != nil && len(*req.Changes.Remove) > 0 {
		for _, removeInterval := range *req.Changes.Remove {
			updatedIntervals = removeTimeInterval(updatedIntervals, removeInterval)
		}
	}

	// Add intervals
	if req.Changes.Add != nil && len(*req.Changes.Add) > 0 {
		updatedIntervals = append(updatedIntervals, *req.Changes.Add...)
		// Re-process to merge overlapping intervals
		chunks, err := convertIntervalsIntoChunks(updatedIntervals)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_intervals",
				"message": "Invalid time intervals after processing changes",
				"details": err.Error(),
			})
			return
		}
		updatedIntervals = groupConsecutiveChunks(chunks)
	}

	// Convert final intervals back to chunks for database storage
	finalChunks, err := convertIntervalsIntoChunks(updatedIntervals)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_intervals",
			"message": "Invalid final time intervals",
			"details": err.Error(),
		})
		return
	}

	// Clear existing availability and insert new availability
	err = replaceAvailability(c.Request.Context(), s.pgxPool, userID, currentUser.OrgID, UserRole(currentUser.Role), finalChunks)
	if err != nil {
		s.logger.Error("Failed to update availability", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to update availability",
		})
		return
	}

	// Log successful update
	s.logger.Info("Availability updated successfully",
		zap.String("user_id", userID),
		zap.String("updated_by", currentUser.UserID),
		zap.Int("final_interval_count", len(finalChunks)),
	)

	// Return updated availability
	response := Availability{
		UserId:                 userID,
		AvailableTimeIntervals: updatedIntervals,
	}

	c.JSON(http.StatusOK, response)
}

func (s *Service) GetBatchAvailability(c *gin.Context) {
	// Validate Firebase token and get user claims
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Parse request body
	var req BatchAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate user IDs provided
	if len(req.UserIds) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "no_users",
			"message": "At least one user ID is required",
		})
		return
	}

	// Non-admin users can only request their own availability
	if currentUser.Role != "admin" {
		// Check if all requested user IDs are the current user
		for _, userID := range req.UserIds {
			if userID != currentUser.UserID {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "forbidden",
					"message": "Can only request your own availability unless you are an admin",
				})
				return
			}
		}
	}

	response := make([]map[string]interface{}, 0, len(req.UserIds))

	for _, userID := range req.UserIds {
		availabilityRecords, err := getAvailability(c.Request.Context(), s.pgxPool, userID)
		if err != nil {
			s.logger.Error("Failed to get batch availability", zap.Error(err), zap.String("user_id", userID))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to retrieve availability",
			})
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

		response = append(response, map[string]interface{}{
			"user_id":                  userID,
			"available_time_intervals": availableTimeIntervals,
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

// replaceAvailability clears existing availability for a user and inserts new availability
func replaceAvailability(ctx context.Context, pgxPool *pgxpool.Pool, userID, orgID string, role UserRole, chunks []TimeInterval) error {
	// Start transaction
	tx, err := pgxPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Delete existing availability for the user
	_, err = tx.Exec(ctx, "DELETE FROM availability WHERE user_id = $1", userID)
	if err != nil {
		return err
	}

	// Insert new availability chunks
	for _, interval := range chunks {
		availabilityID := uuid.New().String()
		_, err = tx.Exec(ctx, queryBatchUpsertAvailabilitySQL,
			availabilityID,
			orgID,
			userID,
			role,
			interval[0],
			interval[1],
			false, // matched
			time.Now(),
			time.Now(),
		)
		if err != nil {
			return err
		}
	}

	// Commit transaction
	return tx.Commit(ctx)
}

// removeTimeInterval removes a specific time interval from a list of intervals
func removeTimeInterval(intervals []TimeInterval, toRemove TimeInterval) []TimeInterval {
	if len(toRemove) != 2 {
		return intervals
	}

	removeStart := toRemove[0]
	removeEnd := toRemove[1]
	result := []TimeInterval{}

	for _, interval := range intervals {
		if len(interval) != 2 {
			continue
		}

		intervalStart := interval[0]
		intervalEnd := interval[1]

		// No overlap - keep the interval
		if intervalEnd.Before(removeStart) || intervalStart.After(removeEnd) {
			result = append(result, interval)
			continue
		}

		// Partial overlap - split the interval
		// Before the removal range
		if intervalStart.Before(removeStart) {
			result = append(result, TimeInterval{intervalStart, removeStart})
		}

		// After the removal range
		if intervalEnd.After(removeEnd) {
			result = append(result, TimeInterval{removeEnd, intervalEnd})
		}

		// If the interval is completely within the removal range, we don't add anything
	}

	return result
}
