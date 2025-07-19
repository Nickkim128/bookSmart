package scheduler

import (
	"github.com/gin-gonic/gin"
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
	// TODO: Implement user retrieval logic
}

func (s *Service) ListUsers(c *gin.Context) {
	// TODO: Implement user listing logic
}

func (s *Service) UpdateUser(c *gin.Context, userID string) {
	// TODO: Implement user update logic
}

func (s *Service) DeleteUser(c *gin.Context, userID string) {
	// TODO: Implement user deletion logic
}
