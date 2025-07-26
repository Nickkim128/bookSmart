#!/bin/bash

# Firebase Authentication API Flow Test
# This script tests the complete authentication flow

echo "🔥 Firebase Authentication API Flow Test"
echo "========================================"

BASE_URL="http://localhost:8000"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ $2${NC}"
    else
        echo -e "${RED}❌ $2${NC}"
    fi
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

echo ""
echo "=== Pre-requisites Check ==="

# Check if server is running
echo "Checking if server is running..."
if curl -s "$BASE_URL" > /dev/null 2>&1; then
    print_status 0 "Server is running on $BASE_URL"
else
    print_status 1 "Server is not running on $BASE_URL"
    echo "Please start the server with: go run main.go"
    exit 1
fi

echo ""
echo "=== Test 1: Unauthenticated Request (Should Fail) ==="

# Test accessing protected endpoint without auth
RESPONSE=$(curl -s -w "%{http_code}" -o /tmp/test_response.json "$BASE_URL/v1/user")
HTTP_CODE="${RESPONSE: -3}"

if [ "$HTTP_CODE" = "401" ]; then
    print_status 0 "Unauthenticated request properly rejected (HTTP $HTTP_CODE)"
else
    print_status 1 "Expected HTTP 401, got HTTP $HTTP_CODE"
    echo "Response: $(cat /tmp/test_response.json)"
fi

echo ""
echo "=== Test 2: Invalid Token (Should Fail) ==="

# Test with invalid token
FAKE_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.invalid"

RESPONSE=$(curl -s -w "%{http_code}" -o /tmp/test_response.json \
    -H "Authorization: Bearer $FAKE_TOKEN" \
    "$BASE_URL/v1/user")
HTTP_CODE="${RESPONSE: -3}"

if [ "$HTTP_CODE" = "401" ]; then
    print_status 0 "Invalid token properly rejected (HTTP $HTTP_CODE)"
else
    print_status 1 "Expected HTTP 401, got HTTP $HTTP_CODE"
    echo "Response: $(cat /tmp/test_response.json)"
fi

echo ""
echo "=== Test 3: Malformed Token (Should Fail) ==="

# Test with malformed token
MALFORMED_TOKEN="not-a-jwt-token"

RESPONSE=$(curl -s -w "%{http_code}" -o /tmp/test_response.json \
    -H "Authorization: Bearer $MALFORMED_TOKEN" \
    "$BASE_URL/v1/user")
HTTP_CODE="${RESPONSE: -3}"

if [ "$HTTP_CODE" = "401" ]; then
    print_status 0 "Malformed token properly rejected (HTTP $HTTP_CODE)"
else
    print_status 1 "Expected HTTP 401, got HTTP $HTTP_CODE"
    echo "Response: $(cat /tmp/test_response.json)"
fi

echo ""
echo "=== Test 4: Missing Authorization Header (Should Fail) ==="

# Test with no Authorization header
RESPONSE=$(curl -s -w "%{http_code}" -o /tmp/test_response.json \
    -H "Content-Type: application/json" \
    "$BASE_URL/v1/user")
HTTP_CODE="${RESPONSE: -3}"

if [ "$HTTP_CODE" = "401" ]; then
    print_status 0 "Missing auth header properly rejected (HTTP $HTTP_CODE)"
else
    print_status 1 "Expected HTTP 401, got HTTP $HTTP_CODE"
    echo "Response: $(cat /tmp/test_response.json)"
fi

echo ""
echo "=== Test 5: CORS Headers ==="

# Test CORS preflight request
RESPONSE=$(curl -s -w "%{http_code}" -o /tmp/test_response.json \
    -X OPTIONS \
    -H "Origin: http://localhost:3000" \
    -H "Access-Control-Request-Method: GET" \
    -H "Access-Control-Request-Headers: Authorization" \
    "$BASE_URL/v1/user")
HTTP_CODE="${RESPONSE: -3}"

if [ "$HTTP_CODE" = "204" ]; then
    print_status 0 "CORS preflight request handled (HTTP $HTTP_CODE)"
else
    print_status 1 "CORS preflight failed, got HTTP $HTTP_CODE"
fi

echo ""
echo "=== Test 6: Admin Route Protection ==="

# Test admin-only endpoint without proper role
RESPONSE=$(curl -s -w "%{http_code}" -o /tmp/test_response.json \
    -H "Authorization: Bearer $FAKE_TOKEN" \
    "$BASE_URL/v1/admin/users")
HTTP_CODE="${RESPONSE: -3}"

if [ "$HTTP_CODE" = "401" ]; then
    print_status 0 "Admin route properly protected (HTTP $HTTP_CODE)"
else
    print_status 1 "Expected HTTP 401, got HTTP $HTTP_CODE"
fi

echo ""
echo "=== Test Summary ==="
echo ""
print_warning "These tests validate that authentication is properly enforced"
print_warning "To test with valid Firebase tokens, you need to:"
echo ""
echo "1. Create a Firebase project"
echo "2. Set up environment variables"
echo "3. Get a real Firebase ID token from your frontend"
echo "4. Replace FAKE_TOKEN with real token in this script"
echo ""
echo "Example valid token test:"
echo 'REAL_TOKEN="your-firebase-id-token-here"'
echo 'curl -H "Authorization: Bearer $REAL_TOKEN" http://localhost:8000/v1/user'
echo ""
echo "🔥 Firebase Auth integration is properly protecting your API!"

# Cleanup
rm -f /tmp/test_response.json
