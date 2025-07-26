package auth

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// User represents the authenticated user context
type User struct {
	UserID       string    `json:"user_id"`
	FirebaseUID  string    `json:"firebase_uid"`
	OrgID        string    `json:"org_id"`
	Role         string    `json:"role"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	LastLoginAt  time.Time `json:"last_login_at"`
	EmailVerified bool     `json:"email_verified"`
}

// AuthMiddleware provides Firebase authentication middleware for Gin
type AuthMiddleware struct {
	firebaseService *FirebaseService
	db              *sql.DB
	logger          *zap.Logger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(firebaseService *FirebaseService, db *sql.DB, logger *zap.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		firebaseService: firebaseService,
		db:              db,
		logger:          logger,
	}
}

// RequireAuth middleware that requires valid Firebase authentication
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := m.extractToken(c)
		if err != nil {
			m.logger.Warn("Authentication failed: invalid token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Valid authentication token required",
			})
			c.Abort()
			return
		}

		// Verify the token with Firebase
		claims, err := m.firebaseService.VerifyIDToken(token)
		if err != nil {
			m.logger.Warn("Authentication failed: token verification failed", 
				zap.Error(err), 
				zap.String("token_prefix", token[:min(10, len(token))]))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized", 
				"message": "Invalid authentication token",
			})
			c.Abort()
			return
		}

		// Look up the user in our database
		user, err := m.getUserByFirebaseUID(claims.UID)
		if err != nil {
			m.logger.Error("Failed to get user from database", 
				zap.Error(err), 
				zap.String("firebase_uid", claims.UID))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Failed to retrieve user information",
			})
			c.Abort()
			return
		}

		if user == nil {
			// User exists in Firebase but not in our database
			// Only allow access to create user endpoint for new users
			if c.Request.Method == "POST" && strings.Contains(c.Request.URL.Path, "/v1/user/") {
				m.logger.Info("New Firebase user accessing create user endpoint", 
					zap.String("firebase_uid", claims.UID),
					zap.String("email", getEmailFromClaims(claims)),
					zap.String("path", c.Request.URL.Path))
				
				// Add Firebase info to context for user creation
				c.Set("firebaseUID", claims.UID)
				c.Set("firebaseToken", claims)
				c.Set("firebaseEmail", getEmailFromClaims(claims))
				c.Set("isNewUser", true)
				c.Next()
				return
			}
			
			// Block access to all other endpoints for new users
			m.logger.Info("New Firebase user blocked from accessing endpoint - user record required", 
				zap.String("firebase_uid", claims.UID),
				zap.String("email", getEmailFromClaims(claims)),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method))
			
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "user_record_required",
				"message": "Please create your user profile first. Visit /v1/user/{user_id} with POST method.",
				"firebase_uid": claims.UID,
				"suggested_endpoint": fmt.Sprintf("/v1/user/%s", claims.UID),
			})
			c.Abort()
			return
		}

		// Update last login time
		if err := m.updateLastLogin(user.UserID); err != nil {
			m.logger.Warn("Failed to update last login time", 
				zap.Error(err), 
				zap.String("user_id", user.UserID))
		}

		// Add user to context
		c.Set("currentUser", user)
		c.Set("firebaseToken", claims)
		
		m.logger.Info("User authenticated successfully", 
			zap.String("user_id", user.UserID),
			zap.String("email", user.Email),
			zap.String("role", user.Role))

		c.Next()
	}
}

// RequireRole middleware that requires specific role(s)
func (m *AuthMiddleware) RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("currentUser")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		currentUser := user.(*User)
		
		// Check if user has one of the allowed roles
		for _, role := range allowedRoles {
			if currentUser.Role == role {
				c.Next()
				return
			}
		}

		m.logger.Warn("Access denied: insufficient role", 
			zap.String("user_id", currentUser.UserID),
			zap.String("user_role", currentUser.Role),
			zap.Strings("required_roles", allowedRoles))

		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Insufficient permissions for this action",
		})
		c.Abort()
	}
}

// RequireOrganization middleware that ensures user belongs to specific organization
func (m *AuthMiddleware) RequireOrganization(orgID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("currentUser")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		currentUser := user.(*User)
		
		if currentUser.OrgID != orgID {
			m.logger.Warn("Access denied: organization mismatch", 
				zap.String("user_id", currentUser.UserID),
				zap.String("user_org", currentUser.OrgID),
				zap.String("required_org", orgID))

			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Access denied for this organization",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// extractToken extracts the Bearer token from the Authorization header
func (m *AuthMiddleware) extractToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header missing")
	}

	// Expected format: "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("empty token")
	}

	return token, nil
}

// getUserByFirebaseUID retrieves user from database using Firebase UID
func (m *AuthMiddleware) getUserByFirebaseUID(firebaseUID string) (*User, error) {
	query := `
		SELECT user_id, firebase_uid, org_id, role, first_name, last_name, 
		       email, COALESCE(last_login_at, created_at) as last_login_at, 
		       COALESCE(email_verified, false) as email_verified
		FROM users 
		WHERE firebase_uid = $1 AND status = 'active'
	`

	var user User
	err := m.db.QueryRow(query, firebaseUID).Scan(
		&user.UserID,
		&user.FirebaseUID,
		&user.OrgID,
		&user.Role,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.LastLoginAt,
		&user.EmailVerified,
	)

	if err == sql.ErrNoRows {
		return nil, nil // User not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	return &user, nil
}

// updateLastLogin updates the user's last login timestamp
func (m *AuthMiddleware) updateLastLogin(userID string) error {
	query := `UPDATE users SET last_login_at = NOW() WHERE user_id = $1`
	_, err := m.db.Exec(query, userID)
	return err
}

// getEmailFromClaims safely extracts email from Firebase token claims
func getEmailFromClaims(token *auth.Token) string {
	if email, ok := token.Claims["email"].(string); ok {
		return email
	}
	return ""
}

// Helper function for min (Go < 1.21 compatibility)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetCurrentUser helper function to get current user from Gin context
func GetCurrentUser(c *gin.Context) (*User, error) {
	user, exists := c.Get("currentUser")
	if !exists {
		return nil, fmt.Errorf("user not found in context")
	}

	currentUser, ok := user.(*User)
	if !ok {
		return nil, fmt.Errorf("invalid user type in context")
	}

	return currentUser, nil
}

// IsAdmin helper function to check if current user is admin
func IsAdmin(c *gin.Context) bool {
	user, err := GetCurrentUser(c)
	if err != nil {
		return false
	}
	return user.Role == "admin"
}

// IsTutor helper function to check if current user is tutor
func IsTutor(c *gin.Context) bool {
	user, err := GetCurrentUser(c)
	if err != nil {
		return false
	}
	return user.Role == "tutor"
}

// IsStudent helper function to check if current user is student  
func IsStudent(c *gin.Context) bool {
	user, err := GetCurrentUser(c)
	if err != nil {
		return false
	}
	return user.Role == "student"
}
