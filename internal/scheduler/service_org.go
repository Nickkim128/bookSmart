package scheduler

import (
	"net/http"
	"scheduler-api/internal/auth"
	"strings"

	_ "embed"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OrgService interface {
	CreateOrg(*gin.Context, string)
	DeleteOrg(*gin.Context, string)
}

var _ OrgService = (*Service)(nil)

//go:embed queries/org/create_organization.sql
var queryCreateOrganizationSQL string

//go:embed queries/org/delete_organization.sql
var queryDeleteOrganizationSQL string

func (s *Service) CreateOrg(c *gin.Context, orgID string) {
	// Validate Firebase token and get user claims
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Only admins can create organizations
	if currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Admin access required to create organizations",
		})
		return
	}

	// Parse request body
	var req struct {
		OrganizationID string `json:"organization_id" binding:"required"`
		Name           string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate organization ID matches path parameter
	if req.OrganizationID != orgID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "id_mismatch",
			"message": "Organization ID in body must match path parameter",
		})
		return
	}

	// Validate UUID format for organization ID
	if _, err := uuid.Parse(orgID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_uuid",
			"message": "Organization ID must be a valid UUID",
		})
		return
	}

	// Validate organization name
	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_name",
			"message": "Organization name cannot be empty",
		})
		return
	}

	if len(req.Name) > 255 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_name",
			"message": "Organization name cannot exceed 255 characters",
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

	// Create organization in database
	var organization struct {
		OrganizationID string `json:"organization_id"`
		Name           string `json:"name"`
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at"`
	}

	err = tx.QueryRow(queryCreateOrganizationSQL, req.OrganizationID, req.Name).Scan(
		&organization.OrganizationID,
		&organization.Name,
		&organization.CreatedAt,
		&organization.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "organization_exists",
				"message": "Organization with this ID already exists",
			})
		} else {
			s.logger.Error("Failed to create organization", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to create organization",
			})
		}
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to save organization",
		})
		return
	}

	// Log successful organization creation
	s.logger.Info("Organization created successfully",
		zap.String("organization_id", organization.OrganizationID),
		zap.String("name", organization.Name),
		zap.String("created_by", currentUser.UserID),
	)

	c.JSON(http.StatusCreated, organization)
}

func (s *Service) DeleteOrg(c *gin.Context, orgID string) {
	// Validate Firebase token and get user claims
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Only admins can delete organizations
	if currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Admin access required to delete organizations",
		})
		return
	}

	// Validate UUID format for organization ID
	if _, err := uuid.Parse(orgID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_uuid",
			"message": "Organization ID must be a valid UUID",
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

	// Check if organization exists and get its details before deletion
	var orgExists int
	var orgName string
	err = tx.QueryRow("SELECT COUNT(*), COALESCE(MAX(name), '') FROM organizations WHERE organization_id = $1", orgID).Scan(&orgExists, &orgName)
	if err != nil {
		s.logger.Error("Failed to check organization existence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to verify organization",
		})
		return
	}

	if orgExists == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "organization_not_found",
			"message": "Organization not found",
		})
		return
	}

	// Get count of related resources for logging
	var userCount, courseCount int
	err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE org_id = $1", orgID).Scan(&userCount)
	if err != nil {
		s.logger.Error("Failed to count users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to verify organization dependencies",
		})
		return
	}

	err = tx.QueryRow("SELECT COUNT(*) FROM courses WHERE org_id = $1", orgID).Scan(&courseCount)
	if err != nil {
		s.logger.Error("Failed to count courses", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to verify organization dependencies",
		})
		return
	}

	// Log the deletion attempt with impact analysis
	s.logger.Warn("Organization deletion requested",
		zap.String("organization_id", orgID),
		zap.String("organization_name", orgName),
		zap.String("requested_by", currentUser.UserID),
		zap.Int("affected_users", userCount),
		zap.Int("affected_courses", courseCount),
	)

	// Perform the deletion (cascades to users and courses due to foreign key constraints)
	result, err := tx.Exec(queryDeleteOrganizationSQL, orgID)
	if err != nil {
		s.logger.Error("Failed to delete organization", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to delete organization",
		})
		return
	}

	// Verify deletion was successful
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error("Failed to get rows affected", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to verify deletion",
		})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "organization_not_found",
			"message": "Organization not found",
		})
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to complete deletion",
		})
		return
	}

	// Log successful deletion
	s.logger.Info("Organization deleted successfully",
		zap.String("organization_id", orgID),
		zap.String("organization_name", orgName),
		zap.String("deleted_by", currentUser.UserID),
		zap.Int("deleted_users", userCount),
		zap.Int("deleted_courses", courseCount),
	)

	c.JSON(http.StatusNoContent, nil)
}
