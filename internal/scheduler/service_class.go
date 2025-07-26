package scheduler

import (
	"context"
	"database/sql"
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

type ClassService interface {
	CreateClass(*gin.Context)
	ListUserClasses(*gin.Context, string)
	ListCourseClasses(*gin.Context, string)
}

var _ ClassService = (*Service)(nil)

func (s *Service) CreateClass(c *gin.Context) {
	// Validate Firebase token and get user claims
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Only admins and tutors can create classes
	if currentUser.Role != "admin" && currentUser.Role != "tutor" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Admin or tutor access required to create classes",
		})
		return
	}

	// Parse request body
	var req Class
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate UUID format for class ID
	if _, err := uuid.Parse(req.ClassId); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_uuid",
			"message": "Class ID must be a valid UUID",
		})
		return
	}

	// Validate required fields
	if req.Duration <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_duration",
			"message": "Duration must be greater than 0 minutes",
		})
		return
	}

	if req.StartTime.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_start_time",
			"message": "Start time is required",
		})
		return
	}

	// Validate start time is not in the past
	if req.StartTime.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_start_time",
			"message": "Start time cannot be in the past",
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

	if len(req.Teachers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_participants",
			"message": "At least one teacher is required",
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

	// Validate all participants exist in the organization
	allParticipants := append(req.Students, req.Teachers...)
	participantCount, err := validateParticipantsExist(tx, allParticipants, currentUser.OrgID)
	if err != nil {
		s.logger.Error("Failed to validate participants", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to validate participants",
		})
		return
	}

	if participantCount != len(allParticipants) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "participant_not_found",
			"message": "One or more participants do not exist or are not in your organization",
		})
		return
	}

	// If course_id is provided, validate it exists
	if req.CourseId != nil && *req.CourseId != "" {
		exists, err := validateCourseExists(tx, *req.CourseId, currentUser.OrgID)
		if err != nil {
			s.logger.Error("Failed to validate course", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to validate course",
			})
			return
		}
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "course_not_found",
				"message": "Specified course does not exist or is not in your organization",
			})
			return
		}
	}

	// Create class in database
	var class struct {
		ClassID   string    `json:"class_id"`
		CourseID  *string   `json:"course_id"`
		StartTime time.Time `json:"start_time"`
		Duration  int       `json:"duration"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	err = tx.QueryRow(queryCreateClassSQL,
		req.ClassId, req.CourseId, req.StartTime, req.Duration, currentUser.OrgID).Scan(
		&class.ClassID, &class.CourseID, &class.StartTime, &class.Duration,
		&class.CreatedAt, &class.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "class_exists",
				"message": "Class with this ID already exists",
			})
		} else {
			s.logger.Error("Failed to create class", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to create class",
			})
		}
		return
	}

	// Add participants to class
	for _, studentID := range req.Students {
		_, err = tx.Exec(queryAddClassParticipantSQL, req.ClassId, studentID, "student")
		if err != nil {
			s.logger.Error("Failed to add student to class", zap.Error(err), zap.String("student_id", studentID))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to add students to class",
			})
			return
		}
	}

	for _, teacherID := range req.Teachers {
		_, err = tx.Exec(queryAddClassParticipantSQL, req.ClassId, teacherID, "teacher")
		if err != nil {
			s.logger.Error("Failed to add teacher to class", zap.Error(err), zap.String("teacher_id", teacherID))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to add teachers to class",
			})
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to save class",
		})
		return
	}

	// Prepare response
	response := Class{
		ClassId:   class.ClassID,
		CourseId:  class.CourseID,
		StartTime: class.StartTime,
		Duration:  class.Duration,
		Students:  req.Students,
		Teachers:  req.Teachers,
	}

	// Log successful class creation
	s.logger.Info("Class created successfully",
		zap.String("class_id", class.ClassID),
		zap.String("created_by", currentUser.UserID),
		zap.Int("student_count", len(req.Students)),
		zap.Int("teacher_count", len(req.Teachers)),
	)

	c.JSON(http.StatusCreated, response)
}

func (s *Service) ListUserClasses(c *gin.Context, userID string) {
	// Validate Firebase token and get user claims
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Users can only view their own classes (unless admin)
	if currentUser.UserID != userID && currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Can only view your own classes",
		})
		return
	}

	// Get classes from database
	classRecords, err := listUserClasses(c.Request.Context(), s.pgxPool, userID)
	if err != nil {
		s.logger.Error("Failed to get user classes", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to retrieve user classes",
		})
		return
	}

	// Enrich classes with participant information
	classes := make([]Class, len(classRecords))
	for i, classRecord := range classRecords {
		// Get participants for this class
		students, teachers, err := getClassParticipants(c.Request.Context(), s.pgxPool, classRecord.ClassId)
		if err != nil {
			s.logger.Error("Failed to get class participants", zap.Error(err), zap.String("class_id", classRecord.ClassId))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to retrieve class participants",
			})
			return
		}

		classes[i] = Class{
			ClassId:   classRecord.ClassId,
			CourseId:  classRecord.CourseId,
			StartTime: classRecord.StartTime,
			Duration:  classRecord.Duration,
			Students:  students,
			Teachers:  teachers,
		}
	}

	c.JSON(http.StatusOK, classes)
}

func (s *Service) ListCourseClasses(c *gin.Context, courseID string) {
	// Validate Firebase token and get user claims
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Validate UUID format for course ID
	if _, err := uuid.Parse(courseID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_uuid",
			"message": "Course ID must be a valid UUID",
		})
		return
	}

	// Validate that the course exists and user has access to it
	hasAccess, err := validateCourseAccess(c.Request.Context(), s.pgxPool, courseID, currentUser.UserID, currentUser.Role, currentUser.OrgID)
	if err != nil {
		s.logger.Error("Failed to validate course access", zap.Error(err), zap.String("course_id", courseID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to validate course access",
		})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "You don't have access to this course",
		})
		return
	}

	// Get classes for the course
	classRecords, err := listCourseClasses(c.Request.Context(), s.pgxPool, courseID)
	if err != nil {
		s.logger.Error("Failed to get course classes", zap.Error(err), zap.String("course_id", courseID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to retrieve course classes",
		})
		return
	}

	// Enrich classes with participant information
	classes := make([]Class, len(classRecords))
	for i, classRecord := range classRecords {
		// Get participants for this class
		students, teachers, err := getClassParticipants(c.Request.Context(), s.pgxPool, classRecord.ClassId)
		if err != nil {
			s.logger.Error("Failed to get class participants", zap.Error(err), zap.String("class_id", classRecord.ClassId))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to retrieve class participants",
			})
			return
		}

		classes[i] = Class{
			ClassId:   classRecord.ClassId,
			CourseId:  classRecord.CourseId,
			StartTime: classRecord.StartTime,
			Duration:  classRecord.Duration,
			Students:  students,
			Teachers:  teachers,
		}
	}

	c.JSON(http.StatusOK, classes)
}

//go:embed queries/class/create_class.sql
var queryCreateClassSQL string

//go:embed queries/class/list_user_classes.sql
var queryListUserClassesSQL string

//go:embed queries/class/list_course_classes.sql
var queryListCourseClassesSQL string

//go:embed queries/class/add_class_participant.sql
var queryAddClassParticipantSQL string

//go:embed queries/class/get_class_participants.sql
var queryGetClassParticipantsSQL string

//go:embed queries/class/validate_course_access.sql
var queryValidateCourseAccessSQL string

// ClassRecord represents a class record from the database
type ClassRecord struct {
	ClassId   string    `db:"class_id"`
	CourseId  *string   `db:"course_id"`
	StartTime time.Time `db:"start_time"`
	Duration  int       `db:"duration"`
}

func listUserClasses(ctx context.Context, pgxPool *pgxpool.Pool, userID string) ([]ClassRecord, error) {
	classes := []ClassRecord{}
	return classes, pgxscan.Select(ctx, pgxPool, &classes, queryListUserClassesSQL, userID)
}

func listCourseClasses(ctx context.Context, pgxPool *pgxpool.Pool, courseID string) ([]ClassRecord, error) {
	classes := []ClassRecord{}
	return classes, pgxscan.Select(ctx, pgxPool, &classes, queryListCourseClassesSQL, courseID)
}

func getClassParticipants(ctx context.Context, pgxPool *pgxpool.Pool, classID string) ([]string, []string, error) {
	var participants []struct {
		UserID string `db:"user_id"`
		Role   string `db:"role"`
	}

	err := pgxscan.Select(ctx, pgxPool, &participants, queryGetClassParticipantsSQL, classID)
	if err != nil {
		return nil, nil, err
	}

	var students, teachers []string
	for _, participant := range participants {
		switch participant.Role {
		case "student":
			students = append(students, participant.UserID)
		case "teacher":
			teachers = append(teachers, participant.UserID)
		}
	}

	return students, teachers, nil
}

func validateCourseAccess(ctx context.Context, pgxPool *pgxpool.Pool, courseID, userID, role, orgID string) (bool, error) {
	// Admins have access to all courses in their organization
	if role == "admin" {
		var exists bool
		err := pgxPool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM courses WHERE course_id = $1 AND org_id = $2)", courseID, orgID).Scan(&exists)
		return exists, err
	}

	// For students and tutors, check if they're enrolled in the course
	var hasAccess bool
	err := pgxPool.QueryRow(ctx, queryValidateCourseAccessSQL, courseID, userID, orgID).Scan(&hasAccess)
	return hasAccess, err
}

func validateParticipantsExist(tx *sql.Tx, userIDs []string, orgID string) (int, error) {
	var count int
	err := tx.QueryRow("SELECT COUNT(*) FROM users WHERE user_id = ANY($1) AND org_id = $2", pq.Array(userIDs), orgID).Scan(&count)
	return count, err
}

func validateCourseExists(tx *sql.Tx, courseID, orgID string) (bool, error) {
	var exists bool
	err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM courses WHERE course_id = $1 AND org_id = $2)", courseID, orgID).Scan(&exists)
	return exists, err
}
