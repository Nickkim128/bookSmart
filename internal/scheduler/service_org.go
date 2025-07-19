package scheduler

import (
	"github.com/gin-gonic/gin"
)

type OrgService interface {
	CreateOrg(*gin.Context, string)
	DeleteOrg(*gin.Context, string)
}

var _ OrgService = (*Service)(nil)

func (s *Service) CreateOrg(c *gin.Context, orgID string) {
	// TODO: Implement organization creation logic
}

func (s *Service) DeleteOrg(c *gin.Context, orgID string) {
	// TODO: Implement organization deletion logic
}
