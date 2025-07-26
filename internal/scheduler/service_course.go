package scheduler

import (
	"context"
	"net/http"
	"scheduler-api/internal/auth"
	"strings"
	"time"

	_ "embed"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

type CourseService interface {
	CreateCourse(*gin.Context)
	GetCourse(*gin.Context, string)
	ListCourses(*gin.Context)
	UpdateCourse(*gin.Context, string)
}

var _ CourseService = (*Service)(nil)

func (s *Service) CreateCourse(c *gin.Context) {
	// Validate Firebase token and get user claims
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Only admins and tutors can create courses
	if currentUser.Role != "admin" && currentUser.Role != "tutor" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Admin or tutor access required to create courses",
		})
		return
	}

	// Parse request body
	var req struct {
		CourseID          string    `json:"course_id" binding:"required"`
		CourseName        string    `json:"course_name" binding:"required"`
		CourseDescription *string   `json:"course_description"`
		Students          []string  `json:"students" binding:"required"`
		Tutors            []string  `json:"tutors" binding:"required"`
		StartAt           time.Time `json:"start_at" binding:"required"`
		EndAt             time.Time `json:"end_at" binding:"required"`
		Interval          string    `json:"interval" binding:"required"`
		Frequency         int       `json:"frequency" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate UUID format for course ID
	if _, err := uuid.Parse(req.CourseID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_uuid",
			"message": "Course ID must be a valid UUID",
		})
		return
	}

	// Validate course name
	req.CourseName = strings.TrimSpace(req.CourseName)
	if len(req.CourseName) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_name",
			"message": "Course name cannot be empty",
		})
		return
	}

	// Validate dates
	if !req.EndAt.After(req.StartAt) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_dates",
			"message": "End date must be after start date",
		})
		return
	}

	// Validate interval
	validIntervals := []string{"weekly", "monthly", "bi-weekly"}
	validInterval := false
	for _, interval := range validIntervals {
		if req.Interval == interval {
			validInterval = true
			break
		}
	}
	if !validInterval {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_interval",
			"message": "Interval must be one of: weekly, monthly, bi-weekly",
		})
		return
	}

	// Validate participants
	if len(req.Students) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_participants",
			"message": "At least one student is required",
		})
		return
	}

	if len(req.Tutors) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_participants",
			"message": "At least one tutor is required",
		})
		return
	}

	// Start database transaction
	tx, err := s.sqlDB.Begin()
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to start transaction",
		})
		return
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			}
		}
	}()

	// Validate all participants exist and are in the same organization
	allParticipants := append(req.Students, req.Tutors...)

	// Validate all users exist
	rows, err := tx.Query(queryCheckUsersExistSQL, pq.Array(allParticipants), currentUser.OrgID)
	if err != nil {
		s.logger.Error("Failed to check users exist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to validate participants",
		})
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Error("Failed to close rows", zap.Error(err))
		}
	}()

	foundUsers := make(map[string]struct{})
	for rows.Next() {
		var userID, role, orgID string
		if err := rows.Scan(&userID, &role, &orgID); err != nil {
			s.logger.Error("Failed to scan user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to validate participants",
			})
			return
		}
		foundUsers[userID] = struct{}{}
	}

	// Check if all participants were found
	for _, userID := range allParticipants {
		if _, found := foundUsers[userID]; !found {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "participant_not_found",
				"message": "One or more participants do not exist or are not in your organization",
			})
			return
		}
	}

	// Create course in database
	var course struct {
		CourseID          string     `json:"course_id"`
		OrgID             string     `json:"org_id"`
		CourseName        string     `json:"course_name"`
		CourseDescription *string    `json:"course_description"`
		StartAt           time.Time  `json:"start_at"`
		EndAt             time.Time  `json:"end_at"`
		Interval          string     `json:"interval"`
		Frequency         int        `json:"frequency"`
		CreatedAt         time.Time  `json:"created_at"`
		UpdatedAt         time.Time  `json:"updated_at"`
	}

	err = tx.QueryRow(queryCreateCourseSQL, 
		req.CourseID, currentUser.OrgID, req.CourseName, req.CourseDescription,
		req.StartAt, req.EndAt, req.Interval, req.Frequency).Scan(
		&course.CourseID, &course.OrgID, &course.CourseName, &course.CourseDescription,
		&course.StartAt, &course.EndAt, &course.Interval, &course.Frequency,
		&course.CreatedAt, &course.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "course_exists",
				"message": "Course with this ID already exists",
			})
		} else {
			s.logger.Error("Failed to create course", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to create course",
			})
		}
		return
	}

	// Add students to course
	for _, studentID := range req.Students {
		_, err = tx.Exec(queryAddCourseStudentsSQL, studentID, req.CourseID)
		if err != nil {
			s.logger.Error("Failed to add student to course", zap.Error(err), zap.String("student_id", studentID))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to add students to course",
			})
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to save course",
		})
		return
	}

	// Add participants to response
	response := gin.H{
		"course_id":          course.CourseID,
		"org_id":             course.OrgID,
		"course_name":        course.CourseName,
		"course_description": course.CourseDescription,
		"students":           req.Students,
		"tutors":             req.Tutors,
		"start_at":           course.StartAt,
		"end_at":             course.EndAt,
		"interval":           course.Interval,
		"frequency":          course.Frequency,
		"created_at":         course.CreatedAt,
		"updated_at":         course.UpdatedAt,
	}

	// Log successful course creation
	s.logger.Info("Course created successfully",
		zap.String("course_id", course.CourseID),
		zap.String("course_name", course.CourseName),
		zap.String("created_by", currentUser.UserID),
		zap.Int("student_count", len(req.Students)),
		zap.Int("tutor_count", len(req.Tutors)),
	)

	c.JSON(http.StatusCreated, response)
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

//go:embed queries/course/create_course.sql
var queryCreateCourseSQL string

//go:embed queries/course/get_course.sql
var queryGetCourseSQL string

//go:embed queries/course/add_course_students.sql
var queryAddCourseStudentsSQL string

//go:embed queries/course/check_users_exist.sql
var queryCheckUsersExistSQL string

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
