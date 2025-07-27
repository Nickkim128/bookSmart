package scheduler

import (
	"context"
	_ "embed"
	"net/http"
	"scheduler-api/internal/auth"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClassService interface {
	CreateClass(*gin.Context)
	ListUserClasses(*gin.Context, string)
	ListCourseClasses(*gin.Context, string)
}

var _ ClassService = (*Service)(nil)

func (s *Service) CreateClass(c *gin.Context) {
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
			"message": "Only admin can create classes",
		})
		return
	}
	createClassRequest := Class{}
	if err := c.ShouldBindJSON(&createClassRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var (
		// Hard coded org id here for now.
		orgID   = "00000000-0000-0000-0000-000000000001"
		classID = uuid.New().String()
		now     = time.Now()
	)

	err = createClass(c.Request.Context(), s.pgxPool, createClassRequest, classID, orgID, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Failed to create Class": err.Error()})
		return
	}

	classParticipantsErr := createClassParticipants(c.Request.Context(), s.pgxPool, createClassRequest, classID, orgID, now)
	if classParticipantsErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Failed to add class participants": classParticipantsErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Class created successfully"})
}

func (s *Service) ListUserClasses(c *gin.Context, userID string) {
	classes, err := listUserClasses(c.Request.Context(), s.pgxPool, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Failed to list user classes": err.Error()})
		return
	}

	c.JSON(http.StatusOK, classes)
}

func (s *Service) ListCourseClasses(c *gin.Context, courseID string) {

}

//go:embed queries/class/list_user_classes.sql
var queryListUserClassesSQL string

//go:embed queries/class/create_class.sql
var createClassSQL string

//go:embed queries/class/create_class_participants.sql
var createClassParticipantsSQL string

func listUserClasses(ctx context.Context, pgxPool *pgxpool.Pool, userID string) ([]Class, error) {
	classes := []Class{}
	return classes, pgxscan.Select(ctx, pgxPool, &classes, queryListUserClassesSQL, userID)
}

func createClass(ctx context.Context, pgxPool *pgxpool.Pool, class Class, classID, orgID string, now time.Time) error {
	_, err := pgxPool.Exec(ctx, createClassSQL, classID, class.CourseId, orgID, class.StartTime, class.Duration, now, now)
	return err
}

func createClassParticipants(ctx context.Context, pgxPool *pgxpool.Pool, class Class, classID string, orgID string, now time.Time) error {
	batch := &pgx.Batch{}

	for _, student := range class.Students {
		batch.Queue(createClassParticipantsSQL, classID, student, "student", now)
	}

	for _, teacher := range class.Teachers {
		batch.Queue(createClassParticipantsSQL, classID, teacher, "teacher", now)
	}

	batchResult := pgxPool.SendBatch(ctx, batch)
	defer func() {
		_ = batchResult.Close()
	}()

	for i := 0; i < len(class.Students)+len(class.Teachers); i++ {
		_, err := batchResult.Exec()
		if err != nil {
			return err
		}
	}

	return nil
}
