package scheduler

import (
	"context"
	_ "embed"
	"net/http"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
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

//go:embed queries/list_courses.sql
var queryListCoursesSQL string

func (s *Service) ListCourses(c *gin.Context) {
	organizationID := "00000000-0000-0000-0000-000000000001"
	courses, err := listCourses(c.Request.Context(), s.pgxPool, organizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, courses)
}

func listCourses(ctx context.Context, pgxPool *pgxpool.Pool, organizationID string) ([]Course, error) {
	courses := []Course{}
	return courses, pgxscan.Select(ctx, pgxPool, &courses, queryListCoursesSQL, organizationID)
}

func (s *Service) UpdateCourse(c *gin.Context, courseID string) {
	// TODO: Implement course update logic
}
