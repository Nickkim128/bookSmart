# Firebase Authentication Integration

This document explains how to set up and use Firebase Authentication with the BookSmart Scheduler API.

## Overview

The API now includes Firebase Authentication integration providing:

- Secure JWT token-based authentication
- Role-based access control (admin, tutor, student)
- Multi-tenant data isolation by organization
- User management with Firebase sync

## Environment Setup

### 1. Firebase Project Configuration

Create a Firebase project and service account:

1. Go to [Firebase Console](https://console.firebase.google.com/)
2. Create a new project or select existing project
3. Go to Project Settings → Service Accounts
4. Generate a new private key (downloads JSON file)
5. Save the JSON file securely

### 2. Environment Variables

Add Firebase configuration to your `.env` file:

```bash
# Option 1: Service Account File (Development)
FIREBASE_SERVICE_ACCOUNT_PATH=./firebase-service-account.json

# Option 2: Individual Variables (Production)
FIREBASE_PROJECT_ID=your-firebase-project-id
FIREBASE_PRIVATE_KEY="your-firebase-private-key-content-here"
FIREBASE_CLIENT_EMAIL=firebase-adminsdk-xyz@your-project-id.iam.gserviceaccount.com
FIREBASE_CLIENT_ID=your-client-id

# Web API Key (for frontend)
FIREBASE_API_KEY=your-web-api-key
```

### 3. Database Migration

Run the database migration to add Firebase authentication fields:

```bash
go run cmd/migrate/main.go -neon
```

This adds:

- `firebase_uid` column to users table
- `email_verified`, `last_login_at`, `status` columns
- Necessary indexes for performance

## Authentication Flow

### 1. User Registration

**Frontend (JavaScript/TypeScript):**

```javascript
import { getAuth, createUserWithEmailAndPassword } from "firebase/auth";

const auth = getAuth();
const userCredential = await createUserWithEmailAndPassword(
  auth,
  email,
  password,
);
const idToken = await userCredential.user.getIdToken();

// Send user details to your API to create database record
const response = await fetch("/v1/users", {
  method: "POST",
  headers: {
    Authorization: `Bearer ${idToken}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    org_id: "organization-uuid",
    role: "student",
    first_name: "John",
    last_name: "Doe",
    email: email,
  }),
});
```

### 2. User Login

**Frontend (JavaScript/TypeScript):**

```javascript
import { getAuth, signInWithEmailAndPassword } from "firebase/auth";

const auth = getAuth();
const userCredential = await signInWithEmailAndPassword(auth, email, password);
const idToken = await userCredential.user.getIdToken();

// Use idToken for subsequent API calls
localStorage.setItem("authToken", idToken);
```

### 3. API Requests

**Frontend (JavaScript/TypeScript):**

```javascript
const token = localStorage.getItem("authToken");

const response = await fetch("/v1/classes", {
  headers: {
    Authorization: `Bearer ${token}`,
    "Content-Type": "application/json",
  },
});
```

## API Usage Examples

### Protected Endpoints

All API endpoints now require authentication. Include the Firebase ID token in the Authorization header:

```bash
curl -H "Authorization: Bearer YOUR_FIREBASE_ID_TOKEN" \
     -H "Content-Type: application/json" \
     http://localhost:8000/v1/users
```

### Role-Based Access

Different endpoints have different role requirements:

**Admin Only:**

```bash
# Create new organization (admin only)
curl -X POST \
     -H "Authorization: Bearer ADMIN_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"name": "New School"}' \
     http://localhost:8000/v1/org/new-org-id
```

**Tutor or Admin:**

```bash
# Create new class (tutors and admins can create classes)
curl -X POST \
     -H "Authorization: Bearer TUTOR_OR_ADMIN_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"course_id": "course-uuid", "start_time": "2025-01-15T10:00:00Z", "duration": 60}' \
     http://localhost:8000/v1/class
```

**Any Authenticated User:**

```bash
# Get own availability (any authenticated user can view their own data)
curl -H "Authorization: Bearer USER_TOKEN" \
     http://localhost:8000/v1/user/USER_ID/availability
```

## Data Isolation

The system enforces organization-based data isolation:

- Users can only see data from their own organization
- API automatically filters results by user's `org_id`
- Cross-organization access is prevented

**Example:** When a tutor from "Bright Minds Tutoring" requests classes, they only see classes from their organization, never from "Excellence Academy".

## User Management

### Creating Users

1. **Create Firebase User:** Use Firebase Auth SDK in frontend
2. **Create Database Record:** Call API with Firebase ID token
3. **Set Custom Claims:** API automatically sets role-based claims

```go
// API automatically sets custom claims
claims := map[string]interface{}{
    "role":   "tutor",
    "org_id": "organization-uuid",
}
firebaseService.SetCustomClaims(firebaseUID, claims)
```

### User Roles

- **admin**: Full access to organization data, can manage users and organizations
- **tutor**: Can create/manage classes, view student information
- **student**: Can view own classes and availability, limited access

### Frontend Integration Example

**React Hook for Authentication:**

```javascript
import { useState, useEffect } from "react";
import { getAuth, onAuthStateChanged } from "firebase/auth";

export function useAuth() {
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const auth = getAuth();
    const unsubscribe = onAuthStateChanged(auth, async (firebaseUser) => {
      if (firebaseUser) {
        const idToken = await firebaseUser.getIdToken();
        setUser(firebaseUser);
        setToken(idToken);
      } else {
        setUser(null);
        setToken(null);
      }
      setLoading(false);
    });

    return unsubscribe;
  }, []);

  return { user, token, loading };
}
```

**API Client with Authentication:**

```javascript
class APIClient {
  constructor(token) {
    this.token = token;
    this.baseURL = process.env.REACT_APP_API_URL || "http://localhost:8000";
  }

  async request(endpoint, options = {}) {
    const url = `${this.baseURL}${endpoint}`;
    const config = {
      headers: {
        Authorization: `Bearer ${this.token}`,
        "Content-Type": "application/json",
        ...options.headers,
      },
      ...options,
    };

    const response = await fetch(url, config);

    if (!response.ok) {
      throw new Error(`API request failed: ${response.statusText}`);
    }

    return response.json();
  }

  async getClasses() {
    return this.request("/v1/class/user");
  }

  async createClass(classData) {
    return this.request("/v1/class", {
      method: "POST",
      body: JSON.stringify(classData),
    });
  }
}
```

## Security Features

### Token Validation

- All tokens are verified with Firebase Admin SDK
- Expired or invalid tokens are rejected
- User context is attached to each request

### Data Protection

- Organization-based data isolation
- Role-based access control
- Audit trail with user tracking

### Error Handling

- Clear error messages for authentication failures
- Secure error responses (no sensitive data leakage)
- Proper HTTP status codes

## Testing

### Manual Testing with curl

1. **Get Firebase ID Token:**
   Use Firebase Auth REST API or frontend to get token

2. **Test Protected Endpoint:**

   ```bash
   curl -H "Authorization: Bearer YOUR_TOKEN" \
        http://localhost:8000/v1/users
   ```

3. **Test Role-Based Access:**

   ```bash
   # Should work with admin token
   curl -H "Authorization: Bearer ADMIN_TOKEN" \
        -X POST http://localhost:8000/v1/admin/users

   # Should fail with student token
   curl -H "Authorization: Bearer STUDENT_TOKEN" \
        -X POST http://localhost:8000/v1/admin/users
   ```

### Integration Testing

```go
func TestAuthentication(t *testing.T) {
    // Test case: Valid token should allow access
    // Test case: Invalid token should return 401
    // Test case: Missing token should return 401
    // Test case: Role-based access control
}
```

## Troubleshooting

### Common Issues

1. **"Invalid authentication token"**
   - Check if Firebase service account is configured correctly
   - Verify token hasn't expired
   - Ensure token format is correct (Bearer TOKEN)

2. **"User account not found"**
   - User exists in Firebase but not in database
   - Create user record in database via API

3. **"Access denied for this organization"**
   - User trying to access data from different organization
   - Check user's org_id in database

4. **"Insufficient permissions"**
   - User role doesn't have required permissions
   - Check role-based access control requirements

### Debug Mode

Enable debug logging for authentication:

```bash
LOG_LEVEL=debug go run main.go
```

This will log:

- Authentication attempts
- Token validation results
- Role-based access decisions
- Database queries for user lookup

## Production Deployment

### Environment Variables

```bash
# Use individual variables instead of file path
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_PRIVATE_KEY="your-firebase-private-key-content-here"
FIREBASE_CLIENT_EMAIL=firebase-adminsdk@your-project.iam.gserviceaccount.com
FIREBASE_CLIENT_ID=your-client-id

# Database
DATABASE_URL=postgresql://user:pass@host:5432/db?sslmode=require

# Security
GO_ENV=production
```

### Security Checklist

- [ ] Firebase service account key is secure
- [ ] Environment variables are properly set
- [ ] Database migrations are applied
- [ ] CORS is configured for your domain
- [ ] HTTPS is enforced in production
- [ ] Logging is configured for monitoring

## Support

For issues with Firebase Authentication integration:

1. Check Firebase Console for authentication logs
2. Review API server logs for authentication errors
3. Verify database schema includes Firebase UID fields
4. Test with curl to isolate frontend vs backend issues
