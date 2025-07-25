package scheduler

import (
	"context"
	_ "embed"
	"net/http"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserService interface {
	CreateUser(*gin.Context, string)
	GetUser(*gin.Context, string)
	ListUsers(*gin.Context)
	UpdateUser(*gin.Context, string)
	DeleteUser(*gin.Context, string)
}

var _ UserService = (*Service)(nil)

func (s *Service) CreateUser(c *gin.Context, userID string) {
	// TODO: Implement user creation logic
}

func (s *Service) GetUser(c *gin.Context, userID string) {
	user, err := getUser(c.Request.Context(), s.pgxPool, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	courses, err := getUserCourses(c.Request.Context(), s.pgxPool, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user.Courses = &courses
	c.JSON(http.StatusOK, user)
}

func (s *Service) ListUsers(c *gin.Context) {
	organizationID := "00000000-0000-0000-0000-000000000001"
	users, err := listUsers(c.Request.Context(), s.pgxPool, organizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range users {
		courses, err := getUserCourses(c.Request.Context(), s.pgxPool, users[i].UserId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		users[i].Courses = &courses
	}

	c.JSON(http.StatusOK, users)
}

func (s *Service) UpdateUser(c *gin.Context, userID string) {
	// TODO: Implement user update logic
}

func (s *Service) DeleteUser(c *gin.Context, userID string) {
	// TODO: Implement user deletion logic
}

//go:embed queries/user/get_user.sql
var queryGetUserSQL string

//go:embed queries/user/list_users.sql
var queryListUsersSQL string

//go:embed queries/user/get_user_courses.sql
var queryGetUserCoursesSQL string

func listUsers(ctx context.Context, pgxPool *pgxpool.Pool, organizationID string) ([]User, error) {
	users := []User{}
	return users, pgxscan.Select(ctx, pgxPool, &users, queryListUsersSQL, organizationID)
}

func getUserCourses(ctx context.Context, pgxPool *pgxpool.Pool, userID string) ([]string, error) {
	courses := []string{}
	return courses, pgxscan.Select(ctx, pgxPool, &courses, queryGetUserCoursesSQL, userID)
}

func getUser(ctx context.Context, pgxPool *pgxpool.Pool, userID string) (User, error) {
	user := User{}
	return user, pgxscan.Get(ctx, pgxPool, &user, queryGetUserSQL, userID)
}
