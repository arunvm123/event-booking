#!/bin/bash

# Simple test script for User Service
# Make sure the service is running on localhost:8081

BASE_URL="http://localhost:8081"

echo "Testing User Service API..."

# Test health check
echo "1. Testing health check..."
curl -X GET "$BASE_URL/health" | jq .
echo -e "\n"

# Test user registration
echo "2. Testing user registration..."
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/users/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "first_name": "Test",
    "last_name": "User"
  }')

echo $REGISTER_RESPONSE | jq .
echo -e "\n"

# Test user login
echo "3. Testing user login..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/users/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }')

echo $LOGIN_RESPONSE | jq .

# Extract token for future use
TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.access_token')
echo -e "\nExtracted JWT Token: $TOKEN\n"

# Test duplicate registration (should fail)
echo "4. Testing duplicate registration (should fail)..."
curl -s -X POST "$BASE_URL/api/users/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "first_name": "Test",
    "last_name": "User"
  }' | jq .
echo -e "\n"

# Test invalid login (should fail)
echo "5. Testing invalid login (should fail)..."
curl -s -X POST "$BASE_URL/api/users/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "wrongpassword"
  }' | jq .
echo -e "\n"

echo "User Service testing completed!"
