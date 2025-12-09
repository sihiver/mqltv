#!/bin/bash

# Create User (30 hari subscription)
echo "=== Create User ==="
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "client01",
    "password": "rahasia123",
    "full_name": "Client Pertama",
    "email": "client01@example.com",
    "max_connections": 2,
    "duration_days": 30,
    "notes": "Paket Premium 2 device"
  }' | jq '.'

echo ""
echo "=== List All Users ==="
curl -s http://localhost:8080/api/users | jq '.[] | {id, username, full_name, expires_at, days_remaining, is_expired, max_connections}'

echo ""
echo "=== Update User (Extend 7 days) ==="
curl -X PUT http://localhost:8080/api/users/1 \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Client Pertama Updated",
    "email": "client01@example.com",
    "max_connections": 3,
    "is_active": true,
    "extend_days": 7,
    "notes": "Upgraded ke 3 device"
  }' | jq '.'

echo ""
echo "=== Get User Connections ==="
curl -s http://localhost:8080/api/users/1/connections | jq '.'

echo ""
echo "=== Reset Password ==="
curl -X POST http://localhost:8080/api/users/1/reset-password \
  -H "Content-Type: application/json" \
  -d '{
    "new_password": "newpassword123"
  }' | jq '.'
