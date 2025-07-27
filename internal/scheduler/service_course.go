package scheduler

import (
	"context"
	_ "embed"
	"net/http"
	"scheduler-api/internal/auth"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
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
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	if currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Only admin can create courses",
		})
		return
	}

	createCourseRequest := Course{}
	if err := c.ShouldBindJSON(&createCourseRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var (
		// Hard coded org id here for now.
		orgID = "00000000-0000-0000-0000-000000000001"
		now   = time.Now()
	)

	err = createCourse(c.Request.Context(), s.pgxPool, createCourseRequest, orgID, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Failed to create Course": err.Error()})
		return
	}

	courseParticipantsError := addCourseParticipants(c.Request.Context(), s.pgxPool, createCourseRequest, now)

	if courseParticipantsError != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Failed to add course participants": courseParticipantsError.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Course created successfully"})
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
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	if currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Only admin can create courses",
		})
		return
	}

	updateRequest := CourseUpdate{}
	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate course ID format
	if courseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Course ID is required"})
		return
	}

	now := time.Now()
	err = updateCourse(c.Request.Context(), s.pgxPool, courseID, updateRequest, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Course updated successfully"})
}

//go:embed queries/course/get_course.sql
var queryGetCourseSQL string

//go:embed queries/course/list_courses.sql
var queryListCoursesSQL string

//go:embed queries/course/get_course_users.sql
var queryGetCourseUsersSQL string

//go:embed queries/course/create_course.sql
var createCourseSql string

//go:embed queries/course/update_course.sql
var updateCourseSQL string

//go:embed queries/course/add_course_participant.sql
var addCourseParticipantSQL string

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

func createCourse(ctx context.Context, pgxPool *pgxpool.Pool, course Course, orgId string, now time.Time) error {
	_, err := pgxPool.Exec(ctx, createCourseSql, course.CourseId, orgId, course.CourseName, course.CourseDescription, course.StartAt, course.EndAt, course.Interval, course.Frequency, now)
	return err
}

func updateCourse(ctx context.Context, pgxPool *pgxpool.Pool, courseID string, update CourseUpdate, now time.Time) error {
	_, err := pgxPool.Exec(ctx, updateCourseSQL, courseID, update.CourseName, nil, update.StartAt, update.EndAt, update.Interval, update.Frequency, now)
	return err
}

func addCourseParticipants(ctx context.Context, pgxPool *pgxpool.Pool, course Course, now time.Time) error {
	batch := &pgx.Batch{}

	for _, student := range course.Students {
		batch.Queue(addCourseParticipantSQL, student, course.CourseId, "student", now)
	}

	for _, tutor := range course.Tutors {
		batch.Queue(addCourseParticipantSQL, tutor, course.CourseId, "teacher", now)
	}

	batchResult := pgxPool.SendBatch(ctx, batch)
	defer func() {
		_ = batchResult.Close()
	}()

	for i := 0; i < len(course.Students)+len(course.Tutors); i++ {
		_, err := batchResult.Exec()
		if err != nil {
			return err
		}
	}

	return nil
}
