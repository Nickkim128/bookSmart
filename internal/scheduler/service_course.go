package scheduler

import (
	"github.com/gin-gonic/gin"
)

type CourseService interface {
	CreateCourse(*gin.Context)
	GetCourse(*gin.Context, string)
	ListCourses(*gin.Context)
	UpdateCourse(*gin.Context, string)
}

var _ CourseService = (*Service)(nil)

func (s *Service) CreateCourse(c *gin.Context) {
	// TODO: Implement course creation logic
}

func (s *Service) GetCourse(c *gin.Context, courseID string) {
	// TODO: Implement course retrieval logic
}

func (s *Service) ListCourses(c *gin.Context) {
	// TODO: Implement course listing logic
}

func (s *Service) UpdateCourse(c *gin.Context, courseID string) {
	// TODO: Implement course update logic
}
