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

//go:embed queries/course/get_course.sql
var queryGetCourseSQL string

func (s *Service) GetCourse(c *gin.Context, courseID string) {
	course, err := getCourse(c.Request.Context(), s.pgxPool, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, course)
}

//go:embed queries/course/list_courses.sql
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

func (s *Service) UpdateCourse(c *gin.Context, courseID string) {
	// TODO: Implement course update logic
}

func listCourses(ctx context.Context, pgxPool *pgxpool.Pool, organizationID string) ([]Course, error) {
	courses := []Course{}
	return courses, pgxscan.Select(ctx, pgxPool, &courses, queryListCoursesSQL, organizationID)
}

func getCourse(ctx context.Context, pgxPool *pgxpool.Pool, courseID string) (Course, error) {
	course := Course{}
	return course, pgxscan.Get(ctx, pgxPool, &course, queryGetCourseSQL, courseID)
}
