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
	course, err := getCourse(c.Request.Context(), s.pgxPool, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	users, err := getCourseUsers(c.Request.Context(), s.pgxPool, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	course.Students = users
	c.JSON(http.StatusOK, course)
}

func (s *Service) ListCourses(c *gin.Context) {
	organizationID := "00000000-0000-0000-0000-000000000001"
	courses, err := listCourses(c.Request.Context(), s.pgxPool, organizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range courses {
		users, err := getCourseUsers(c.Request.Context(), s.pgxPool, courses[i].CourseId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		courses[i].Students = users
	}

	c.JSON(http.StatusOK, courses)
}

func (s *Service) UpdateCourse(c *gin.Context, courseID string) {
	// TODO: Implement course update logic
}

//go:embed queries/course/get_course.sql
var queryGetCourseSQL string

//go:embed queries/course/list_courses.sql
var queryListCoursesSQL string

//go:embed queries/course/get_course_users.sql
var queryGetCourseUsersSQL string

func listCourses(ctx context.Context, pgxPool *pgxpool.Pool, organizationID string) ([]Course, error) {
	courses := []Course{}
	return courses, pgxscan.Select(ctx, pgxPool, &courses, queryListCoursesSQL, organizationID)
}

func getCourseUsers(ctx context.Context, pgxPool *pgxpool.Pool, courseID string) ([]string, error) {
	users := []string{}
	return users, pgxscan.Select(ctx, pgxPool, &users, queryGetCourseUsersSQL, courseID)
}

func getCourse(ctx context.Context, pgxPool *pgxpool.Pool, courseID string) (Course, error) {
	course := Course{}
	return course, pgxscan.Get(ctx, pgxPool, &course, queryGetCourseSQL, courseID)
}
