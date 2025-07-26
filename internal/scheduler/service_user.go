package scheduler

import (
	"fmt"
	"net/http"
	"scheduler-api/internal/auth"
	"strings"

	"context"
	_ "embed"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
	// Check if this is a new Firebase user
	isNewUser, exists := c.Get("isNewUser")
	if exists && isNewUser.(bool) {
		// Parse request body
		var req struct {
			OrgID     string `json:"org_id" binding:"required"`
			Role      string `json:"role" binding:"required"`
			FirstName string `json:"first_name" binding:"required"`
			LastName  string `json:"last_name" binding:"required"`
			Email     string `json:"email" binding:"required,email"`
		}
		
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "Invalid request body",
				"details": err.Error(),
			})
			return
		}

		// Validate role
		validRoles := []string{"admin", "student", "tutor"}
		isValidRole := false
		for _, role := range validRoles {
			if req.Role == role {
				isValidRole = true
				break
			}
		}
		if !isValidRole {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_role",
				"message": "Role must be one of: admin, student, tutor",
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

		// Set custom claims for role-based access
		claims := map[string]interface{}{
			"role":   req.Role,
			"org_id": req.OrgID,
		}
		if err := s.firebaseService.SetCustomClaims(userID, claims); err != nil {
			s.logger.Error("Failed to set custom claims", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "firebase_error", 
				"message": "Failed to set user permissions",
			})
			return
		}

		// Create user in database
		query := `
			INSERT INTO users (org_id, firebase_uid, role, first_name, last_name, email, status, email_verified)
			VALUES ($1, $2, $3, $4, $5, $6, 'active', false)
			RETURNING user_id, created_at, updated_at
		`
		
		var dbUserID string
		var createdAt, updatedAt string
		err = tx.QueryRow(query, req.OrgID, userID, req.Role, req.FirstName, req.LastName, req.Email).Scan(
			&dbUserID, &createdAt, &updatedAt)
		if err != nil {
			s.logger.Error("Failed to create user in database", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to create user record",
			})
			return
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			s.logger.Error("Failed to commit transaction", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to save user",
			})
			return
		}

		// Return created user
		user := gin.H{
			"user_id":        dbUserID,
			"firebase_uid":   userID,
			"org_id":         req.OrgID,
			"role":           req.Role,
			"first_name":     req.FirstName,
			"last_name":      req.LastName,
			"email":          req.Email,
			"email_verified": false,
			"status":         "active",
			"created_at":     createdAt,
			"updated_at":     updatedAt,
		}

		s.logger.Info("User created successfully", 
			zap.String("user_id", dbUserID),
			zap.String("firebase_uid", userID),
			zap.String("email", req.Email),
			zap.String("role", req.Role))

		c.JSON(http.StatusCreated, user)
		return
	}

	// For existing users, this would be a different flow
	c.JSON(http.StatusBadRequest, gin.H{
		"error":   "invalid_operation",
		"message": "User creation only allowed for new Firebase users",
	})
}

func (s *Service) GetUser(c *gin.Context, userID string) {
	// Get current user from auth middleware
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Users can only view their own profile (unless admin)
	if currentUser.UserID != userID && currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Can only view your own profile",
		})
		return
	}

	// Query user from database
	query := `
		SELECT user_id, org_id, role, first_name, last_name, email, 
		       COALESCE(email_verified, false) as email_verified,
		       COALESCE(status, 'active') as status,
		       created_at, updated_at, last_login_at
		FROM users 
		WHERE user_id = $1
	`

	var user struct {
		UserID        string     `json:"user_id"`
		OrgID         string     `json:"org_id"`
		Role          string     `json:"role"`
		FirstName     string     `json:"first_name"`
		LastName      string     `json:"last_name"`
		Email         string     `json:"email"`
		EmailVerified bool       `json:"email_verified"`
		Status        string     `json:"status"`
		CreatedAt     string     `json:"created_at"`
		UpdatedAt     string     `json:"updated_at"`
		LastLoginAt   *string    `json:"last_login_at,omitempty"`
	}

	err = s.sqlDB.QueryRow(query, userID).Scan(
		&user.UserID,
		&user.OrgID,
		&user.Role,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.EmailVerified,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "user_not_found",
				"message": "User not found",
			})
		} else {
			s.logger.Error("Failed to query user", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to retrieve user",
			})
		}
		return
	}

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
	// Get current user from auth middleware
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Users can only update their own profile (unless admin)
	if currentUser.UserID != userID && currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Can only update your own profile",
		})
		return
	}

	var req struct {
		FirstName string `json:"first_name,omitempty"`
		LastName  string `json:"last_name,omitempty"`
		Email     string `json:"email,omitempty"`
		Role      string `json:"role,omitempty"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
		return
	}

	// Validate role if provided and check authorization
	if req.Role != "" {
		// Check if role is valid
		validRoles := []string{"admin", "student", "tutor"}
		isValidRole := false
		for _, role := range validRoles {
			if req.Role == role {
				isValidRole = true
				break
			}
		}
		if !isValidRole {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_role",
				"message": "Role must be one of: admin, student, tutor",
			})
			return
		}

		// Security check: Only admins can change roles
		if currentUser.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Admin access required to change user roles",
			})
			return
		}

		// Prevent users from changing their own role
		if currentUser.UserID == userID {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Cannot change your own role",
			})
			return
		}

		// Get current user role from database to check if they're the last admin
		var currentUserRole string
		err := s.sqlDB.QueryRow("SELECT role FROM users WHERE user_id = $1", userID).Scan(&currentUserRole)
		if err != nil {
			s.logger.Error("Failed to get current user role", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to validate role change",
			})
			return
		}

		// If demoting an admin, ensure at least one admin will remain
		if currentUserRole == "admin" && req.Role != "admin" {
			var adminCount int
			err := s.sqlDB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin' AND user_id != $1", userID).Scan(&adminCount)
			if err != nil {
				s.logger.Error("Failed to count admins", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "database_error",
					"message": "Failed to validate admin count",
				})
				return
			}

			if adminCount == 0 {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "forbidden",
					"message": "Cannot demote the last admin user",
				})
				return
			}
		}

		// Log role change attempt
		s.logger.Info("Role change attempted",
			zap.String("admin_user_id", currentUser.UserID),
			zap.String("target_user_id", userID),
			zap.String("old_role", currentUserRole),
			zap.String("new_role", req.Role),
		)
	}

	// Start transaction
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

	// Build dynamic update query
	setParts := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	if req.FirstName != "" {
		setParts = append(setParts, fmt.Sprintf("first_name = $%d", argIndex))
		args = append(args, req.FirstName)
		argIndex++
	}
	if req.LastName != "" {
		setParts = append(setParts, fmt.Sprintf("last_name = $%d", argIndex))
		args = append(args, req.LastName)
		argIndex++
	}
	if req.Email != "" {
		setParts = append(setParts, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, req.Email)
		argIndex++
	}
	if req.Role != "" {
		setParts = append(setParts, fmt.Sprintf("role = $%d", argIndex))
		args = append(args, req.Role)
		argIndex++
	}

	if len(setParts) == 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "no_changes",
			"message": "No fields provided to update",
		})
		return
	}

	// Update database
	query := fmt.Sprintf("UPDATE users SET %s WHERE user_id = $%d", 
		strings.Join(setParts, ", "), argIndex)
	args = append(args, userID)

	_, err = tx.Exec(query, args...)
	if err != nil {
		s.logger.Error("Failed to update user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to update user",
		})
		return
	}

	// Update Firebase custom claims if role changed
	if req.Role != "" {
		// Get Firebase UID
		var firebaseUID string
		err = s.sqlDB.QueryRow("SELECT firebase_uid FROM users WHERE user_id = $1", userID).Scan(&firebaseUID)
		if err != nil {
			s.logger.Error("Failed to get Firebase UID", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "database_error",
				"message": "Failed to get user Firebase ID",
			})
			return
		}

		// Security safeguards implemented above - only admins can change roles
		// and users cannot change their own role or demote the last admin
		claims := map[string]interface{}{
			"role":   req.Role,
			"org_id": currentUser.OrgID,
		}
		
		// Log successful role change
		s.logger.Info("Role change successful",
			zap.String("admin_user_id", currentUser.UserID),
			zap.String("target_user_id", userID),
			zap.String("new_role", req.Role),
			zap.String("firebase_uid", firebaseUID),
		)
		if err := s.firebaseService.SetCustomClaims(firebaseUID, claims); err != nil {
			s.logger.Error("Failed to update custom claims", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "firebase_error",
				"message": "Failed to update user permissions",
			})
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "database_error",
			"message": "Failed to save changes",
		})
		return
	}

	// Return updated user (call GetUser to get fresh data)
	s.GetUser(c, userID)
}

func (s *Service) DeleteUser(c *gin.Context, userID string) {
	// Only admins can delete users
	if !auth.IsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Admin access required",
		})
		return
	}

	// TODO: Implement user deletion
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "not_implemented",
		"message": "User deletion not yet implemented",
	})
}

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
