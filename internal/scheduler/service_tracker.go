package scheduler

import (
	"github.com/gin-gonic/gin"
)

type TrackerService interface {
	GetTrackers(*gin.Context)
}

var _ TrackerService = (*Service)(nil)

func (s *Service) GetTrackers(c *gin.Context) {
	// TODO: Implement course trackers retrieval logic
}
