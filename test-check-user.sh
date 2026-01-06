#!/bin/bash

echo "=== Testing Check User API ==="
echo ""

# Get session cookie first (using correct admin password)
echo "1. Login as admin..."
RESPONSE=$(curl -s -c /tmp/cookies.txt -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"459922"}')

echo "$RESPONSE" | jq '.'

COOKIE=$(grep "admin-session" /tmp/cookies.txt | awk '{print $7}')

if [ -z "$COOKIE" ]; then
  echo "❌ Failed to login"
  exit 1
fi
echo "✅ Login successful"
echo ""

# Check user nizam
echo "2. Testing check user API without password (user: nizam)..."
curl -s -X GET "http://localhost:8080/api/users/check/nizam" \
  -H "Cookie: admin-session=$COOKIE" | jq '.'
echo ""

echo "3. Testing check user API with valid password (user: nizam)..."
curl -s -X GET "http://localhost:8080/api/users/check/nizam?password=123" \
  -H "Cookie: admin-session=$COOKIE" | jq '.'
echo ""

echo "4. Testing check user API with invalid password (user: nizam)..."
curl -s -X GET "http://localhost:8080/api/users/check/nizam?password=wrongpass" \
  -H "Cookie: admin-session=$COOKIE" | jq '.'
echo ""

echo "5. Testing check user API with non-existent user..."
curl -s -X GET "http://localhost:8080/api/users/check/nonexistent" \
  -H "Cookie: admin-session=$COOKIE" | jq '.'
echo ""

echo "=== Test Complete ==="
