package scheduler

import (
	"github.com/gin-gonic/gin"
)

type AvailabilityService interface {
	CreateAvailability(*gin.Context, string)
	GetAvailability(*gin.Context, string)
	UpdateAvailability(*gin.Context, string)
	GetBatchAvailability(*gin.Context)
}

var _ AvailabilityService = (*Service)(nil)

func (s *Service) CreateAvailability(c *gin.Context, userID string) {
	// TODO: Implement availability creation logic
	// - Get user ID from context
	// - Validate request body
	// - Create availability record
	// - Return created availability
}

func (s *Service) GetAvailability(c *gin.Context, userID string) {
	// TODO: Implement availability retrieval logic
	// - Get user ID from context
	// - Query database for availability
	// - Return availability record
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
