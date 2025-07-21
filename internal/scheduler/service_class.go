package scheduler

import (
	"context"
	_ "embed"
	"net/http"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClassService interface {
	CreateClass(*gin.Context)
	ListUserClasses(*gin.Context, string)
	ListCourseClasses(*gin.Context, string)
}

var _ ClassService = (*Service)(nil)

func (s *Service) CreateClass(c *gin.Context) {
	// TODO: Implement class creation logic
}

func (s *Service) ListUserClasses(c *gin.Context, userID string) {
	userID = "10000000-0000-0000-0000-000000000005"
	classes, err := listUserClasses(c.Request.Context(), s.pgxPool, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, classes)
}

func (s *Service) ListCourseClasses(c *gin.Context, courseID string) {

}

//go:embed queries/class/list_user_classes.sql
var queryListUserClassesSQL string

func listUserClasses(ctx context.Context, pgxPool *pgxpool.Pool, userID string) ([]Class, error) {
	classes := []Class{}
	return classes, pgxscan.Select(ctx, pgxPool, &classes, queryListUserClassesSQL, userID)
}
