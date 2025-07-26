package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// FirebaseService handles Firebase authentication operations
type FirebaseService struct {
	client *auth.Client
	ctx    context.Context
}

// FirebaseConfig holds Firebase configuration
type FirebaseConfig struct {
	ProjectID     string `json:"project_id"`
	PrivateKeyID  string `json:"private_key_id"`
	PrivateKey    string `json:"private_key"`
	ClientEmail   string `json:"client_email"`
	ClientID      string `json:"client_id"`
	AuthURI       string `json:"auth_uri"`
	TokenURI      string `json:"token_uri"`
	Type          string `json:"type"`
	Universe      string `json:"universe_domain,omitempty"`
}

// NewFirebaseService creates a new Firebase service instance
func NewFirebaseService() (*FirebaseService, error) {
	ctx := context.Background()

	// Check if we have a service account key file
	serviceAccountPath := os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH")
	if serviceAccountPath != "" {
		return newFirebaseFromFile(ctx, serviceAccountPath)
	}

	// Check if we have service account JSON in environment
	serviceAccountJSON := os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON")
	if serviceAccountJSON != "" {
		return newFirebaseFromJSON(ctx, serviceAccountJSON)
	}

	// Try to use individual environment variables
	return newFirebaseFromEnvVars(ctx)
}

// newFirebaseFromFile initializes Firebase from a service account file
func newFirebaseFromFile(ctx context.Context, filePath string) (*FirebaseService, error) {
	opt := option.WithCredentialsFile(filePath)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase app from file: %w", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Firebase auth client: %w", err)
	}

	return &FirebaseService{
		client: client,
		ctx:    ctx,
	}, nil
}

// newFirebaseFromJSON initializes Firebase from JSON string
func newFirebaseFromJSON(ctx context.Context, serviceAccountJSON string) (*FirebaseService, error) {
	opt := option.WithCredentialsJSON([]byte(serviceAccountJSON))
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase app from JSON: %w", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Firebase auth client: %w", err)
	}

	return &FirebaseService{
		client: client,
		ctx:    ctx,
	}, nil
}

// newFirebaseFromEnvVars initializes Firebase from individual environment variables
func newFirebaseFromEnvVars(ctx context.Context) (*FirebaseService, error) {
	config := FirebaseConfig{
		Type:         "service_account",
		ProjectID:    os.Getenv("FIREBASE_PROJECT_ID"),
		PrivateKeyID: os.Getenv("FIREBASE_PRIVATE_KEY_ID"),
		PrivateKey:   os.Getenv("FIREBASE_PRIVATE_KEY"),
		ClientEmail:  os.Getenv("FIREBASE_CLIENT_EMAIL"),
		ClientID:     os.Getenv("FIREBASE_CLIENT_ID"),
		AuthURI:      "https://accounts.google.com/o/oauth2/auth",
		TokenURI:     "https://oauth2.googleapis.com/token",
		Universe:     "googleapis.com",
	}

	// Validate required fields
	if config.ProjectID == "" || config.PrivateKey == "" || config.ClientEmail == "" {
		return nil, fmt.Errorf("missing required Firebase environment variables: FIREBASE_PROJECT_ID, FIREBASE_PRIVATE_KEY, FIREBASE_CLIENT_EMAIL")
	}

	// Convert to JSON for the SDK
	credentialsJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Firebase credentials: %w", err)
	}

	return newFirebaseFromJSON(ctx, string(credentialsJSON))
}

// VerifyIDToken verifies a Firebase ID token and returns the token claims
func (fs *FirebaseService) VerifyIDToken(idToken string) (*auth.Token, error) {
	token, err := fs.client.VerifyIDToken(fs.ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}
	return token, nil
}

// GetUser gets user information by Firebase UID
func (fs *FirebaseService) GetUser(uid string) (*auth.UserRecord, error) {
	user, err := fs.client.GetUser(fs.ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user %s: %w", uid, err)
	}
	return user, nil
}

// GetUserByEmail gets user information by email
func (fs *FirebaseService) GetUserByEmail(email string) (*auth.UserRecord, error) {
	user, err := fs.client.GetUserByEmail(fs.ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email %s: %w", email, err)
	}
	return user, nil
}

// CreateUser creates a new Firebase user
func (fs *FirebaseService) CreateUser(email, password string) (*auth.UserRecord, error) {
	params := &auth.UserToCreate{}
	params.Email(email).Password(password).EmailVerified(false)

	user, err := fs.client.CreateUser(fs.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

// UpdateUser updates an existing Firebase user
func (fs *FirebaseService) UpdateUser(uid string, email string) (*auth.UserRecord, error) {
	params := &auth.UserToUpdate{}
	params.Email(email)

	user, err := fs.client.UpdateUser(fs.ctx, uid, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update user %s: %w", uid, err)
	}
	return user, nil
}

// DeleteUser deletes a Firebase user
func (fs *FirebaseService) DeleteUser(uid string) error {
	err := fs.client.DeleteUser(fs.ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to delete user %s: %w", uid, err)
	}
	return nil
}

// SetCustomClaims sets custom claims for a user (for role-based access)
func (fs *FirebaseService) SetCustomClaims(uid string, claims map[string]interface{}) error {
	err := fs.client.SetCustomUserClaims(fs.ctx, uid, claims)
	if err != nil {
		return fmt.Errorf("failed to set custom claims for user %s: %w", uid, err)
	}
	return nil
}
