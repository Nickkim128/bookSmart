package scheduler

import (
	"github.com/gin-gonic/gin"
)

type ClassService interface {
	CreateClass(*gin.Context)
	ListUserClasses(*gin.Context)
	ListCourseClasses(*gin.Context)
}

var _ ClassService = (*Service)(nil)

func (s *Service) CreateClass(c *gin.Context) {
	// TODO: Implement class creation logic
}

func (s *Service) ListUserClasses(c *gin.Context) {
	// TODO: Implement user classes retrieval logic
}

func (s *Service) ListCourseClasses(c *gin.Context) {
	// TODO: Implement course classes retrieval logic
}
